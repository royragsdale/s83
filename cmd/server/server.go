package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/royragsdale/s83"
)

type Server struct {
	store               *Store
	difficultyFactor    float64
	difficultyThreshold uint64 // TODO: simplify into Difficulty type to keep in sync
	ttl                 int    // days
	blockList           map[string]bool
	creator             s83.Creator
}

func (srv *Server) handler(w http.ResponseWriter, req *http.Request) {
	// Log requests (TODO: configurable verbosity)
	log.Printf("%s %s %s", req.RemoteAddr, req.Method, req.URL)

	// Servers must support preflight OPTIONS requests to all endpoints
	if req.Method == http.MethodOptions {
		srv.handleOptions(w, req)
		return
	}

	// Common headers
	w.Header().Set("Spring-Version", s83.SpringVersion)
	w.Header().Set("Content-Type", "text/html;charset=utf-8")

	// GET / ("homepage"/difficulty)
	if req.URL.Path == "/" {
		if req.Method != http.MethodGet {
			http.Error(w, "405 - Method Not Allowed: use GET", http.StatusMethodNotAllowed)
			return
		}
		srv.handleDifficulty(w, req)
		return
	}

	// TODO: disagree with SPEC (prevents just normal web browser from opening links)
	// ref: https://github.com/robinsloan/spring-83-spec/issues/7
	/*
		// Check this is an actual Spring-83 client
		if req.Header.Get("Spring-Version") != s83.SpringVersion {
			http.Error(w, "400 - Invalid Spring-Version", http.StatusBadRequest)
			return
		}
	*/

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
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Modified-Since, Spring-Signature, Spring-Version")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Last-Modified, Spring-Difficulty, Spring-Signature, Spring-Version")
	w.WriteHeader(http.StatusNoContent)
}

func (srv *Server) handleDifficulty(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Spring-Difficulty", fmt.Sprintf("%f", srv.difficultyFactor))

	// TODO: insert stats/difficulty factor
	fmt.Fprintf(w, greet)

}

func (srv *Server) handleGetBoard(w http.ResponseWriter, req *http.Request, key string) {
	var board s83.Board
	var err error

	// special case
	// "an ever-changing board...with a timestamp set to the time of the request."
	if key == s83.TestPublic {
		rand.Seed(time.Now().Unix())
		randMsg := magic8Ball[rand.Intn(len(magic8Ball))]
		content := fmt.Sprintf(testBoard, randMsg)
		board, err = srv.creator.NewBoard([]byte(content))
		if err != nil {
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
		log.Println("removing exipred board", board.Publisher)
		srv.store.removeBoard(board)
		http.Error(w, "404 - Board not found", http.StatusNotFound)
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

	skipDifficultyCheck := false
	existingBoard, err := srv.store.boardFromKey(key)
	// there was a valid existing board
	if err == nil {
		if !board.AfterBoard(existingBoard) {
			http.Error(w, "409 - Submission older than existing board", http.StatusConflict)
			return
		}
		// existing boards are grandfathered
		skipDifficultyCheck = true
	}

	// reject boards older than TTL
	if srv.boardExpired(board) {
		http.Error(w, "409 - Submission older than server TTL", http.StatusConflict)
		return
	}

	// check difficulty
	if !skipDifficultyCheck && board.Publisher.Strength() >= srv.difficultyThreshold {
		http.Error(w, "403: Board was submitted for a key that does not meet the difficulty factor", http.StatusForbidden)
		return

	}

	err = srv.store.saveBoard(board)
	if err != nil {
		fmt.Println("error saving board", err)
		http.Error(w, "500 - Internal Server Error", http.StatusInternalServerError)
	} else {
		srv.store.NumBoards += 1
	}

	// TODO: queue board up for gossip
}

func main() {
	// TODO: configure from ENV/file
	blockList := map[string]bool{s83.TestPublic: true}
	difficultyFactor := 0.0
	ttl := 22
	storePath := "store"
	host := ""
	port := 8080
	envPort := os.Getenv("PORT")
	if envPort != "" {
		p, err := strconv.Atoi(envPort)
		if err != nil {
			log.Fatalf("failed parsing PORT: %v\n", err)
		}
		port = p
	}

	store, err := loadStore(storePath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d boards from store %s", store.NumBoards, store.Dir)

	// creator for the test key board
	creator, err := s83.NewCreatorFromKey(s83.TestPrivate)
	if err != nil {
		log.Fatal(err)
	}

	// pre compute difficultyThreshold
	threshold, err := s83.DifficultyThreshold(difficultyFactor)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: NewServer(store)
	srv := &Server{store, difficultyFactor, threshold, ttl, blockList, creator}

	http.HandleFunc("/", srv.handler)

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("server started on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

const greet = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>s83d</title>
</head>
<body>
 <h1>&lt;arbitrary HTML greeting&gt;></h1>
</body>
</html>
`

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

var magic8Ball = []string{
	"It is certain.",
	"It is decidedly so.",
	"Without a doubt.",
	"Yes definitely.",
	"You may rely on it.",
	"As I see it, yes.",
	"Most likely.",
	"Outlook good.",
	"Yes.",
	"Signs point to yes.",
	"Reply hazy, try again.",
	"Ask again later.",
	"Better not tell you now.",
	"Cannot predict now.",
	"Concentrate and ask again.",
	"Don't count on it.",
	"My reply is no.",
	"My sources say no.",
	"Outlook not so good.",
	"Very doubtful. ",
}
