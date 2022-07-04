package main

import (
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/royragsdale/s83"
)

//go:embed static/favicon.ico
var favicon []byte

//go:embed templates/*
var resources embed.FS

const tIndex = "index.html.tmpl"

type indexData struct {
	HeaderLeft  template.HTML
	HeaderRight template.HTML
	Boards      []s83.Board
	ClientCSS   template.CSS
	ClientCSP   template.HTML
	Nonce       string
	Favicon     string
}

func nonce() (string, error) {
	nonceLen := 32
	b := make([]byte, nonceLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// TODO: clean up follower/single board edge case
func (config Config) renderBoards(newBoards map[string]s83.Board, localBoards map[string]s83.Board, follows []s83.Follow, outF *os.File) {

	boards := []s83.Board{}

	// simple Follow based ordering (merge new and local boards)
	for _, f := range follows {
		key := f.Key()
		var b s83.Board
		b, ok := newBoards[key]
		if !ok {
			b, ok = localBoards[key]
			if !ok {
				// neither a new board nor a local board
				continue
			}
		}
		boards = append(boards, b)
	}

	nonce, err := nonce()
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}

	csp := fmt.Sprintf(`default-src 'none';
		style-src 'self' 'unsafe-inline';
		font-src 'self';
		form-action *;
		connect-src *;
		script-src 'nonce-%s';`, nonce)

	data := indexData{
		template.HTML(time.Now().Format("3:04PM<br>Mon, 02 Jan 2006")),
		template.HTML(fmt.Sprintf("%d new<br>(%s)", len(newBoards), config.Name)),
		boards,
		s83.ClientCSS,
		template.HTML(csp),
		nonce,
		config.Favicon,
	}

	err = config.templates.ExecuteTemplate(outF, tIndex, data)
	if err != nil {
		log.Printf("error: %v\n", err)
	}

}

func (c Config) outPath() string {
	fName := time.Now().Format("daily-spring-2006-01-02T15:04:05.html")
	return filepath.Join(c.DataPath(), fName)
}
