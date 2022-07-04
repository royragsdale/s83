package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"regexp"
	"time"

	"github.com/royragsdale/s83"
)

// convenience for error handling
// ref: https://go.dev/blog/error-handling-and-go
type srvHandler func(http.ResponseWriter, *http.Request) error

// satisfy http.Handler
func (fn srvHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: add request IP address
	if err := fn(w, r); err != nil {
		// intentionally thrown error (e.g. bad requests)
		if serr, ok := err.(*srvError); ok {
			if serr.LogError != nil {
				log.Printf("%s: %v\n", serr.Error(), serr.LogError)
			}
			http.Error(w, serr.Error(), serr.Code)
		} else {
			// always log unexpected internal errors
			log.Printf("internal error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (srv *Server) address() string {
	return fmt.Sprintf("%s:%d", srv.host, srv.port)
}

func (srv *Server) handler(w http.ResponseWriter, req *http.Request) error {
	// Log requests (TODO: configurable verbosity)
	log.Printf("%s %s %s", req.RemoteAddr, req.Method, req.URL)

	// Common headers
	w.Header().Set("Spring-Version", s83.SpringVersion)
	w.Header().Set("Content-Type", "text/html;charset=utf-8")

	// Servers must add the appropriate CORS headers to all responses:
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Modified-Since, Spring-Signature, Spring-Version")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Last-Modified, Spring-Signature, Spring-Version")

	// Servers must support preflight OPTIONS requests to all endpoints
	if req.Method == http.MethodOptions {
		return srv.handleOptions(w, req)
	}

	// GET / ("homepage")
	if req.URL.Path == "/" {
		if req.Method != http.MethodGet {
			return newHTTPError(http.StatusMethodNotAllowed, "use GET")
		}
		return srv.handleHome(w, req)
	}

	// GET/PUT /<key> (boards)
	reKey := regexp.MustCompile(`^\/([0-9A-Fa-f]{64}?)$`)
	submatch := reKey.FindStringSubmatch(req.URL.Path)
	if submatch != nil && len(submatch) == 2 {
		key := submatch[1]

		if req.Method == http.MethodGet {
			return srv.handleGetBoard(w, req, key)
		} else if req.Method == http.MethodPut {
			return srv.handlePutBoard(w, req, key)
		} else {
			return newHTTPError(http.StatusMethodNotAllowed, "use GET/PUT")
		}
	}

	// fallthrough failcase
	return newHTTPError(http.StatusBadRequest, "invalid key")
}

func (srv *Server) handleOptions(w http.ResponseWriter, req *http.Request) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type indexData struct {
	Title      string
	NumBoards  int
	TTL        int
	AdminBoard *s83.Board
	TestBoard  *s83.Board
	ClientCSS  template.CSS
}

func (srv *Server) handleHome(w http.ResponseWriter, req *http.Request) error {

	var adminBoard *s83.Board = nil
	if srv.admin != nil {
		if a, err := srv.store.boardFromKey(srv.admin.String()); err == nil {
			adminBoard = &a
		} else {
			log.Println("error loading admin board for homepage")
		}
	}

	var testBoard *s83.Board = nil
	if t, err := srv.testBoard(); err == nil {
		testBoard = &t
	} else {
		log.Println("error loading test board for homepage")
	}

	data := indexData{
		srv.title,
		srv.store.NumBoards,
		srv.ttl,
		adminBoard,
		testBoard,
		s83.ClientCSS,
	}

	return srv.templates.ExecuteTemplate(w, tIndex, data)
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

func (srv *Server) handleGetBoard(w http.ResponseWriter, req *http.Request, key string) error {
	var board s83.Board
	var err error

	if srv.blocked(key) {
		return newHTTPError(http.StatusForbidden, "key blocked")
	}

	// special case
	// "an ever-changing board...with a timestamp set to the time of the request."
	if key == s83.TestPublic {
		board, err = srv.testBoard()
		if err != nil {
			return newHTTPErrorLog(http.StatusInternalServerError, "failed generating board", err)
		}
	} else {
		board, err = srv.store.boardFromKey(key)
		if err != nil {
			// TODO: other errors (internal like)
			return newHTTPError(http.StatusNotFound, "board not found")
		}
	}

	// TODO: handle "tombstone" boards, "404 Not Found"

	if !board.VerifySignature() {
		return newHTTPErrorLog(http.StatusInternalServerError, "bad board", fmt.Errorf("board from store failed signature validation: %s", board.Publisher))
	}

	if srv.boardExpired(board) {
		log.Println("removing expired board", board.Publisher)
		srv.store.removeBoard(board)
		srv.store.NumBoards -= 1 // TODO: store should keep track
		return newHTTPError(http.StatusNotFound, "board not found")
	}

	// <date and time in UTC, RFC 5322 format> TODO: ???
	modTimeStr := req.Header.Get("If-Modified-Since")
	modTime, err := mail.ParseDate(modTimeStr)
	if err == nil && !board.After(modTime) {
		// TODO: improve logging
		// parsed a header and board is not newer than the request. Not Modified.
		log.Printf("304 - board (%s) not newer than request (%s)\n", board.Timestamp(), modTimeStr)
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	// TODO: (optional) special case wrap boards from requests missing a Spring-Version header

	w.Header().Set("Spring-Signature", board.Signature())
	// DO NOT "format" board content. It is user supplied.
	w.Write(board.Content)
	return nil
}

func (srv *Server) blocked(key string) bool {
	_, blocked := srv.blockList[key]
	return blocked
}

func (srv *Server) boardExpired(board s83.Board) bool {
	return !board.After(time.Now().UTC().AddDate(0, 0, -srv.ttl))
}

func (srv *Server) handlePutBoard(w http.ResponseWriter, req *http.Request, key string) error {

	if srv.blocked(key) {
		return newHTTPErrorLog(http.StatusForbidden, "key blocked", fmt.Errorf("PUT blocked for key: %s", key))
	}

	// Validate Board (size, signature, timestamp)
	board, err := s83.BoardFromHTTP(key, req.Header.Get("Spring-Signature"), req.Body)
	if err != nil {
		// TODO: handle 400/401/409/513
		// 400: Board was submitted with improper meta timestamp tags.
		// 401: Board was submitted without a valid signature.
		// 413: Board is larger than 2217 bytes.
		return newHTTPErrorLog(http.StatusBadRequest, "bad board", fmt.Errorf("PUT invalid board for key: %s : %w", key, err))
	}

	boardUpdate := false
	existingBoard, err := srv.store.boardFromKey(key)
	// there was a valid existing board
	if err == nil {
		if !board.AfterBoard(existingBoard) {
			return newHTTPError(http.StatusConflict, "not newer than existing board")
		}
		// existing boards are grandfathered
		boardUpdate = true
	}

	// reject boards older than TTL
	if srv.boardExpired(board) {
		return newHTTPError(http.StatusConflict, fmt.Sprintf("older than TTL: %d days", srv.ttl))
	}

	err = srv.store.saveBoard(board)
	if err != nil {
		fmt.Println("error saving board", err)
		return newHTTPErrorLog(http.StatusInternalServerError, "", fmt.Errorf("error saving board for key: %s : %w", key, err))
	} else if !boardUpdate {
		// TODO: store should keep track
		// only increment if it is a new (previously unseen) board
		srv.store.NumBoards += 1
	}

	// TODO: queue board up for gossip

	// success
	return nil
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
	http.Handle("/", srvHandler(srv.handler))

	log.Printf("starting server on %s", srv.address())
	log.Fatal(http.ListenAndServe(srv.address(), nil))
}
