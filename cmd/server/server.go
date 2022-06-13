package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/royragsdale/s83"
)

func handler(w http.ResponseWriter, req *http.Request) {
	// Common headers
	w.Header().Set("Spring-Version", s83.SpringVersion)
	w.Header().Set("Content-Type", "text/html;charset=utf-8")

	// Check this is an actual Spring-83 client
	if req.Header.Get("Spring-Version") != s83.SpringVersion {
		http.Error(w, "400 - Invalid Spring-Version", http.StatusBadRequest)
		return
	}

	// GET / ("homepage"/difficulty)
	if req.URL.Path == "/" {
		if req.Method != http.MethodGet {
			http.Error(w, "405 - Method Not Allowed: use GET", http.StatusMethodNotAllowed)
			return
		}
		handleDifficulty(w, req)
		return
	}

	// GET/PUT /<key> (boards)
	reKey := regexp.MustCompile(`^\/([0-9A-Fa-f]{64}?)$`)
	submatch := reKey.FindStringSubmatch(req.URL.Path)
	if submatch != nil && len(submatch) == 2 {
		key := submatch[1]

		if req.Method == http.MethodGet {
			handleGetBoard(w, req, key)
			return
		} else if req.Method == http.MethodPut {
			handlePutBoard(w, req, key)
			return
		} else {
			http.Error(w, "405 - Method Not Allowed: use GET/PUT", http.StatusMethodNotAllowed)
			return
		}
	}

	// fallthrough failcase
	http.Error(w, "400 - Bad Request", http.StatusBadRequest)
}

func handleDifficulty(w http.ResponseWriter, req *http.Request) {

	// TODO: load numBoards
	numBoards := 8_500_000
	difficultyFactor := s83.DifficultyFactor(numBoards)
	w.Header().Set("Spring-Difficulty", fmt.Sprintf("%f", difficultyFactor))

	// TODO: insert stats/difficulty factor
	fmt.Fprintf(w, greet)

}

func handleGetBoard(w http.ResponseWriter, req *http.Request, key string) {
	fmt.Fprintf(w, "TODO: GET board: %s\n", key)
}

func handlePutBoard(w http.ResponseWriter, req *http.Request, key string) {

	// fast fail
	// "client must include the publishing timestamp in the If-Unmodified-Since header"
	modSinceHead, err := http.ParseTime(req.Header.Get("If-Modified-Since"))
	if err != nil || modSinceHead.After(time.Now()) {
		http.Error(w, "400 - Invalid If-Modified-Since", http.StatusBadRequest)
		return
	}

	// Authorization
	sig, err := parseAuthorization(req)
	if err != nil {
		log.Println(err, req.Header.Get("Authorization"))
		http.Error(w, "401 - Invalid Authorization Header", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "500 - Failed reading request", http.StatusInternalServerError)
	}

	// Validate Board (size, signature, timestamp)
	board, err := s83.NewBoard(key, sig, body)
	if err != nil {
		// TODO: handle 400/401/409/513
		// 400: Board was submitted with impromper meta timestamp tags.
		// 401: Board was submitted without a valid signature.
		// 513: Board is larger than 2217 bytes.
		http.Error(w, "400 - Bad Board", http.StatusBadRequest)
	}

	// TODO: load previous board from store
	// 403: Board was submitted for a key that does not meet the difficulty factor.
	// 404: No board for this key found on this server.
	// 409: Board was submitted with a timestamp older than the server's timestamp for this key.
	fmt.Fprintf(w, "TODO: PUT board: %s\n", key)
	fmt.Fprintf(w, "%s\n", board)

}

func parseAuthorization(req *http.Request) (s83.Signature, error) {
	//Authorization: Spring-83 Signature=<signature>
	auth := req.Header.Get("Authorization")
	reSig := regexp.MustCompile(`^Spring-83 Signature=([0-9A-Fa-f]{128}?)$`)
	submatch := reSig.FindStringSubmatch(auth)
	if submatch == nil || len(submatch) != 2 {
		return []byte{}, errors.New("Failed to match 'Spring-83 Signature' auth")
	}
	sig, err := hex.DecodeString(submatch[1])
	if err != nil {
		return []byte{}, err
	}
	return sig, nil
}

func main() {
	/*
		GET /<key>
		PUT /<key>
		GET /
	*/
	http.HandleFunc("/", handler)

	host := ""
	port := 8080
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
