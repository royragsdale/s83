package main

import (
	"fmt"
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

func TestPutBoardHandler(t *testing.T) {

	// TODO: add utility function to set up test store
	dir := t.TempDir()
	store, err := loadStore(dir)
	if err != nil {
		t.Fatalf(`An empty directory store should be valid: %v`, err)
	}
	blockList := map[string]bool{s83.TestPublic: true, dateToKey(time.Now()): true}
	srv := &Server{store, 0.0, blockList}

	// test blocklist
	for key, _ := range blockList {
		req, err := http.NewRequest("PUT", "/"+key, nil)
		if err != nil {
			t.Fatal(err)
		}
		// TODO: add utility function to set this up
		req.Header.Set("Content-Type", "text/html;charset=utf-8")
		req.Header.Set("Spring-Version", s83.SpringVersion)
		req.Header.Set("Spring-Signature", "XXX")

		rr := httptest.NewRecorder()
		// TODO: simplify to just srv.handlePutBoard
		handler := http.HandlerFunc(srv.handler)
		handler.ServeHTTP(rr, req)
		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code for denylisted key: got %v want %v",
				status, http.StatusUnauthorized)
		}
	}
}
