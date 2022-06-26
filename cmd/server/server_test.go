package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/royragsdale/s83"
)

func dateToKey(t time.Time) string {
	stub := strings.Repeat("a", s83.KeyLen-7) // must be hex char
	prefix := "83e"                           // valid prefix
	return fmt.Sprintf("%s%s%02d%s", stub, prefix, int(t.Month()), strconv.Itoa(t.Year())[2:])
}

func NewRequest(method string, url string, body io.Reader, t *testing.T) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "text/html;charset=utf-8")
	req.Header.Set("Spring-Version", s83.SpringVersion)
	req.Header.Set("Spring-Signature", "XXX")
	return req
}

func TestPutBoardHandler(t *testing.T) {

	// TODO: add utility function to set up test store
	dir := t.TempDir()
	store, err := loadStore(dir)
	if err != nil {
		t.Fatalf(`An empty directory store should be valid: %v`, err)
	}
	blockList := map[string]bool{s83.TestPublic: true, dateToKey(time.Now()): true}
	srv := &Server{store, 0.0, s83.MaxKey, 22, blockList, s83.Creator{}}

	// test blocklist
	for key, _ := range blockList {
		req := NewRequest("PUT", "/"+key, nil, t)
		rr := httptest.NewRecorder()
		putFunc := func(w http.ResponseWriter, req *http.Request) { srv.handlePutBoard(w, req, key) }
		handler := http.HandlerFunc(putFunc)
		handler.ServeHTTP(rr, req)
		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code for denylisted key: got %v want %v",
				status, http.StatusUnauthorized)
		}
	}

}

// TODO: test boards with format string special charachters to ensure we are
// NEVER formatting board content
