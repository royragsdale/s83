package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/royragsdale/s83"
)

/*
Boards are stored as flat files on disk named `<key>.s83`/. The first line of
the file is the signature. Everything else is the content.
*/

type Store struct {
	Dir       string
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
	pattern := filepath.Join(s.Dir, "*.s83")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, boardPath := range matches {
		//TODO: instead of discarding fill cache
		_, err := s83.BoardFromPath(boardPath)
		if err != nil {
			log.Printf("[warn] bad board at %s: %v\n", filepath.Base(boardPath), err)
		} else {
			s.NumBoards += 1
		}
	}

	return nil
}

func (s *Store) boardFromKey(key string) (s83.Board, error) {
	return s83.BoardFromPath(s.keyToPath(key))
}

func (s *Store) keyToPath(key string) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s.s83", key))
}

func (s *Store) boardPath(b s83.Board) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s.s83", b.Publisher.String()))
}

func (s *Store) saveBoard(b s83.Board) error {
	return b.Save(s.Dir)
}

func (s *Store) removeBoard(b s83.Board) error {
	return os.Remove(s.boardPath(b))
}
