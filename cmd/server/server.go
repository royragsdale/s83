package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"regexp"
	"time"

	"github.com/royragsdale/s83"
)

func (srv *Server) address() string {
	return fmt.Sprintf("%s:%d", srv.host, srv.port)
}

func (srv *Server) handler(w http.ResponseWriter, req *http.Request) {
	// Log requests (TODO: configurable verbosity)
	log.Printf("%s %s %s", req.RemoteAddr, req.Method, req.URL)

	// Common headers
	w.Header().Set("Spring-Version", s83.SpringVersion)
	w.Header().Set("Content-Type", "text/html;charset=utf-8")

	// CORS (TODO: verify details of which requests require it)
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Modified-Since, Spring-Signature, Spring-Version")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Last-Modified, Spring-Difficulty, Spring-Signature, Spring-Version")

	// Servers must support preflight OPTIONS requests to all endpoints
	if req.Method == http.MethodOptions {
		srv.handleOptions(w, req)
		return
	}

	// GET / ("homepage"/difficulty)
	if req.URL.Path == "/" {
		if req.Method != http.MethodGet {
			http.Error(w, "405 - Method Not Allowed: use GET", http.StatusMethodNotAllowed)
			return
		}
		srv.handleDifficulty(w, req)
		return
	}

	// GET/PUT /<key> (boards)
	reKey := regexp.MustCompile(`^\/([0-9A-Fa-f]{64}?)$`)
	submatch := reKey.FindStringSubmatch(req.URL.Path)
	if submatch != nil && len(submatch) == 2 {
		key := submatch[1]

		if req.Method == http.MethodGet {
			srv.handleGetBoard(w, req, key)
			return
		} else if req.Method == http.MethodPut {
			srv.handlePutBoard(w, req, key)
			return
		} else {
			http.Error(w, "405 - Method Not Allowed: use GET/PUT", http.StatusMethodNotAllowed)
			return
		}
	}

	// fallthrough failcase
	http.Error(w, "400 - Bad Request", http.StatusBadRequest)
}

func (srv *Server) handleOptions(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type indexData struct {
	Title      string
	Admin      *s83.Publisher
	NumBoards  int
	TTL        int
	Difficulty float64
}

func (srv *Server) handleDifficulty(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Spring-Difficulty", fmt.Sprintf("%f", srv.difficultyFactor))

	data := indexData{
		srv.title,
		srv.admin,
		srv.store.NumBoards,
		srv.ttl,
		srv.difficultyFactor,
	}

	srv.templates.ExecuteTemplate(w, tIndex, data)
}

type testData struct {
	Color   string
	Message string
	Time    string
}

func (srv *Server) testBoard() (s83.Board, error) {
	// get some fun randomness
	rand.Seed(time.Now().Unix())
	randMsg := magic8Ball[rand.Intn(len(magic8Ball))]
	randColor := colors[rand.Intn(len(colors))]
	data := testData{randColor, randMsg, time.Now().UTC().Format(time.RFC1123)}

	// execute template
	var buf bytes.Buffer
	content := make([]byte, s83.MaxBoardLen)
	srv.templates.ExecuteTemplate(&buf, tTest, data)
	n, err := buf.Read(content)
	if err != nil {
		return s83.Board{}, err
	}

	// create a board from it
	return srv.testCreator.NewBoard(content[:n])

}

func (srv *Server) handleGetBoard(w http.ResponseWriter, req *http.Request, key string) {
	var board s83.Board
	var err error

	// special case
	// "an ever-changing board...with a timestamp set to the time of the request."
	if key == s83.TestPublic {

		board, err = srv.testBoard()
		if err != nil {
			log.Println("500: failed creating test board:", err)
			http.Error(w, "500 - Failed generating board", http.StatusInternalServerError)
			return
		}
	} else {
		board, err = srv.store.boardFromKey(key)
		if err != nil {
			// TODO: other errors (internal like)
			http.Error(w, "404 - Board not found", http.StatusNotFound)
			return
		}
	}

	if !board.VerifySignature() {
		log.Println("loaded board with a failed signature", board)
		http.Error(w, "500 - Bad board", http.StatusInternalServerError)
		return
	}

	if srv.boardExpired(board) {
		log.Println("removing expired board", board.Publisher)
		srv.store.removeBoard(board)
		srv.store.NumBoards -= 1 // TODO: store should keep track
		http.Error(w, "404 - Board not found", http.StatusNotFound)
		return
	}

	// <date and time in UTC, RFC 5322 format> TODO: ???
	modTimeStr := req.Header.Get("If-Modified-Since")
	modTime, err := mail.ParseDate(modTimeStr)
	if err == nil && !board.After(modTime) {
		// TODO: improve logging
		// parsed a header and board is not newer than the request. Not Modified.
		log.Printf("304 - board (%s) not newer than request (%s)\n", board.Timestamp(), modTimeStr)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Spring-Signature", board.Signature())
	// DO NOT "format" board content. It is user supplied.
	w.Write(board.Content)
}

func (srv *Server) blocked(key string) bool {
	_, blocked := srv.blockList[key]
	return blocked
}

func (srv *Server) boardExpired(board s83.Board) bool {
	return !board.After(time.Now().UTC().AddDate(0, 0, -srv.ttl))
}

func (srv *Server) handlePutBoard(w http.ResponseWriter, req *http.Request, key string) {

	if srv.blocked(key) {
		http.Error(w, "401 - Unauthorized", http.StatusUnauthorized)
	}

	// Validate Board (size, signature, timestamp)
	board, err := s83.BoardFromHTTP(key, req.Header.Get("Spring-Signature"), req.Body)
	if err != nil {
		// TODO: handle 400/401/409/513
		// 400: Board was submitted with impromper meta timestamp tags.
		// 401: Board was submitted without a valid signature.
		// 413: Board is larger than 2217 bytes.
		log.Println(err)
		http.Error(w, "400 - Bad Board", http.StatusBadRequest)
		return
	}

	boardUpdate := false
	existingBoard, err := srv.store.boardFromKey(key)
	// there was a valid existing board
	if err == nil {
		if !board.AfterBoard(existingBoard) {
			http.Error(w, "409 - Submission older than existing board", http.StatusConflict)
			return
		}
		// existing boards are grandfathered
		boardUpdate = true
	}

	// reject boards older than TTL
	if srv.boardExpired(board) {
		http.Error(w, "409 - Submission older than server TTL", http.StatusConflict)
		return
	}

	// check difficulty
	if !boardUpdate && board.Publisher.Strength() >= srv.difficultyThreshold {
		http.Error(w, "403: Board was submitted for a key that does not meet the difficulty factor", http.StatusForbidden)
		return

	}

	err = srv.store.saveBoard(board)
	if err != nil {
		fmt.Println("error saving board", err)
		http.Error(w, "500 - Internal Server Error", http.StatusInternalServerError)
	} else if !boardUpdate {
		// TODO: store should keep track
		// only increment if it is a new (previously unseen) board
		srv.store.NumBoards += 1
	}

	// TODO: queue board up for gossip
}

func (srv *Server) favicon(w http.ResponseWriter, r *http.Request) {
	w.Write(favicon)
}

func main() {
	// support just the default -h/--help to describe the environment variables supported
	flag.Usage = envUsage
	flag.Parse()

	srv := NewServerFromEnv()

	http.HandleFunc("/favicon.ico", srv.favicon)

	// all API endpoints
	http.HandleFunc("/", srv.handler)

	log.Printf("starting server on %s", srv.address())
	log.Fatal(http.ListenAndServe(srv.address(), nil))
}

const testBoard = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>s83d | Hello World</title>
</head>
<body>
  <h1>Magic s83-ball</h1>
  <p>%s</p>
</body>
</html>
`
