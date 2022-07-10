// Package store implements a simple flat file backed persistence mechanism for
// storing Spring '83 boards. This store should be suitable for both clients
// and servers. It aims for simplicity and portability over performance or
// features.
//
// On disk a board is simply a plain text file named according to the
// publisher's key. The first line of the file is the signature. Everything
// else is the content.
//
// The store package implements a write through cache that allows read-heavy
// use cases (like a server, where boards are read much more frequently then
// they are updated) to primarily operate out of memory.
package store

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/royragsdale/s83"
)

// allow for future variations with different versions
const ext = ".s83"

type Cache map[string]s83.Board

type Store struct {
	dir       string
	numBoards int
	cache     Cache
}

// New takes a path to a directory on disk and initializes the backing
// data structures. In loading the directory it validates any existing boards
// that are found. NewStore will error if the path provided is not a directory.
func New(path string) (*Store, error) {

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

	store := &Store{absPath, 0, Cache{}}

	return store, store.validate()
}

// validate walks the store directory and checks all the boards
func (s *Store) validate() error {
	pattern := filepath.Join(s.dir, "*"+ext)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, boardPath := range matches {
		key := strings.TrimSuffix(filepath.Base(boardPath), ext)
		b, err := s.Get(key)
		if err == nil {
			s.numBoards += 1
			s.cache[key] = b
		}
	}

	return nil
}

// TODO: provide specific errors like "board not found"

// Get retrieves a board from disk based on the key. On success it returns a
// valid board. Otherwise it returns an appropriate error, either based on the
// store (e.g. the board does not exist) or based on the contents of the file
// (e.g. it is an invalid board, the signature fails to verify, etc).
func (s *Store) Get(key string) (s83.Board, error) {
	// check cache first
	if b, ok := s.cache[key]; ok {
		return b, nil
	}

	data, err := os.ReadFile(s.keyToPath(key))
	if err != nil {
		return s83.Board{}, err
	}
	sigEnd := bytes.Index(data, []byte("\n"))
	if sigEnd != s83.SigLen {
		return s83.Board{}, fmt.Errorf("Invalid signature length: %d", sigEnd)
	}

	// first line stores the signature
	sig, err := hex.DecodeString(string(data[:sigEnd]))
	if err != nil {
		return s83.Board{}, err
	}
	// everything else is content
	content := data[sigEnd+1:]

	// validate on creation
	b, err := s83.NewBoard(key, sig, content)
	if err == nil {
		// valid board, add to cache
		s.cache[key] = b
	}

	return b, err
}

// TODO: consider a variation that keeps a history of boards.

// Add stores a board to disk. This will clobber any existing boards. This
// matches the ephemeral nature of the protocol. Any errors opening or writing
// the backing file will be returned.
func (s *Store) Add(b s83.Board) error {
	overwrite := s.boardExists(b)
	data := append([]byte(b.Signature()+"\n"), b.Content...)
	err := os.WriteFile(s.boardToPath(b), data, 0600)
	if err == nil {
		// successfully saved to disk so update cache
		s.cache[b.Key()] = b

		if !overwrite {
			s.numBoards += 1
		}
	}
	return err
}

// Remove deletes a board from disk based on key. If the board does not exist
// in the store this will return an error.
func (s *Store) Remove(key string) error {

	// proactively remove from cache
	delete(s.cache, key)

	err := os.Remove(s.keyToPath(key))
	if err == nil {
		s.numBoards -= 1
	}
	return err
}

// Count returns the number of boards currently tracked by the store.
func (s *Store) Count() int {
	return s.numBoards
}

/* Convenience functions. */

func (s *Store) boardExists(b s83.Board) bool {
	_, err := os.Stat(s.boardToPath(b))
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) keyToPath(key string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s%s", key, ext))
}

func (s *Store) boardToPath(b s83.Board) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s%s", b.Publisher.String(), ext))
}
