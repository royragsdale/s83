package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/royragsdale/s83"
)

/*
Boards are stored as flat files on disk named `<key>.s83`/. The first line of
the file is the signature. Everything else is the content.
*/

type Store struct {
	Path      string
	NumBoards int
	//Cache     map[string]s83.Board
}

func loadStore(path string) (*Store, error) {

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	st, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	} else if !st.IsDir() {
		return nil, errors.New(fmt.Sprintf("store path (%s) is not a directory", absPath))
	}

	// TODO: count boards
	store := &Store{absPath, 0}

	return store, store.validate()
}

// validate walks the store directory and checks all the boards
func (s *Store) validate() error {
	pattern := filepath.Join(s.Path, "*.s83")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, boardPath := range matches {
		//TODO: instead of discarding fill cache
		_, err := s.boardFromPath(boardPath)
		if err != nil {
			log.Printf("[warn] bad board at %s: %v\n", filepath.Base(boardPath), err)
		} else {
			s.NumBoards += 1
		}
	}

	return nil
}

func (s *Store) boardFromKey(key string) (s83.Board, error) {
	return s.boardFromPath(s.keyToPath(key))
}

func (s *Store) boardFromPath(path string) (s83.Board, error) {

	// TODO: validate path is in store
	data, err := os.ReadFile(path)
	if err != nil {
		return s83.Board{}, err
	}
	line := bytes.Index(data, []byte("\n"))

	// first line stores the signature
	sig, err := hex.DecodeString(string(data[:line]))
	if err != nil {
		return s83.Board{}, err
	}
	// everything else is content
	content := data[line+1:]

	// validate on creation
	return s83.NewBoard(pathToKey(path), sig, content)
}

func (s *Store) saveBoard(board s83.Board) error {
	path := s.keyToPath(board.Publisher.String())
	data := append([]byte(board.Signature()+"\n"), board.Content...)
	return os.WriteFile(path, data, 0600)
}

func (s *Store) keyToPath(key string) string {
	return filepath.Join(s.Path, fmt.Sprintf("%s.s83", key))
}

func pathToKey(path string) string {
	// extract publisher key from file name
	return strings.TrimSuffix(filepath.Base(path), ".s83")
}
