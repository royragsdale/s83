package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/royragsdale/s83"
)

var testBytes = []byte("test content")

func emptyTestStore(t *testing.T) (*Store, error) {
	dir := t.TempDir()
	return New(dir)
}

func testBoard(content []byte) (s83.Board, error) {
	c, err := s83.NewCreatorFromKey(s83.TestPrivate)
	if err != nil {
		return s83.Board{}, err
	}

	b, err := c.NewBoard(testBytes)
	if err != nil {
		return s83.Board{}, err
	}
	return b, nil
}

func TestNew(t *testing.T) {

	store, err := emptyTestStore(t)
	if err != nil {
		t.Errorf(`An empty directory store should be valid: %v`, err)
	}

	if store.numBoards != 0 {
		t.Errorf("An empty directory should have 0 boards")
	}

	fPath := filepath.Join(store.dir, "f")
	if err := os.WriteFile(fPath, testBytes, 0644); err != nil {
		t.Errorf(`Failed setting up test file: %v`, err)
	}

	if _, err = New(fPath); err == nil {
		t.Errorf("A file is not a valid store, should error")
	}

	// TODO: test loading invlid boards
}

func TestNewWithContents(t *testing.T) {

	dir := t.TempDir()

	// test a directory with non board content
	path := filepath.Join(dir, "junk")
	if err := os.WriteFile(path, testBytes, 0644); err != nil {
		t.Fatalf(`Error seeding directory with junk content: %v`, err)
	}

	store, err := New(dir)
	if err != nil {
		t.Errorf(`A directory with invalid content should still load: %v`, err)
	}

	if store.Count() != 0 {
		t.Errorf("junk should not count as content")
	}

	// test a directory with a bad board
	path = filepath.Join(dir, "bad_board"+ext)
	if err := os.WriteFile(path, testBytes, 0644); err != nil {
		t.Fatalf(`Error seeding directory with bad board: %v`, err)
	}

	store, err = New(dir)
	if err != nil {
		t.Errorf(`A directory with bad boards should still load: %v`, err)
	}

	if store.Count() != 0 {
		t.Errorf("bad boards should not count as content")
	}

	// now save a real board there and reload
	b, err := testBoard(testBytes)
	if err != nil {
		t.Fatalf(`Failed creating a test board: %v`, err)
	}
	if err := store.Add(b); err != nil {
		t.Errorf(`error adding valid board: %v`, err)
	}

	// reload store
	store, err = New(dir)
	if err != nil {
		t.Errorf(`A directory with junk/bad/real boards should still load: %v`, err)
	}

	if store.Count() != 1 {
		t.Errorf("real board should count")
	}

	_, err = store.Get(b.Key())
	if err != nil {
		t.Errorf("failure getting existing board")
	}
}

func TestAddBoard(t *testing.T) {

	store, err := emptyTestStore(t)
	if err != nil {
		t.Fatalf(`An empty directory store should be valid: %v`, err)
	}

	b, err := testBoard(testBytes)
	if err != nil {
		t.Fatalf(`Failure making test board: %v`, err)
	}

	if err = store.Add(b); err != nil {
		t.Errorf("error saving valid board")
	}

	if store.numBoards != 1 {
		t.Errorf("failed to increment board count")
	}
	if store.numBoards != store.Count() {
		t.Errorf("count should always match the number of boards")
	}

	if err = store.Add(b); err != nil {
		t.Errorf("error saving over board")
	}

	if store.numBoards != 1 {
		t.Errorf("overwrites should not increment count")
	}

	fromDisk, err := store.Get(b.Key())
	if err != nil {
		t.Errorf("failure getting board we just saved")
	}

	if !fromDisk.Eq(b) {
		t.Errorf("board from disk did not match board pre-save")
	}
}

func TestRemoveBoard(t *testing.T) {
	store, err := emptyTestStore(t)
	if err != nil {
		t.Fatalf(`An empty directory store should be valid: %v`, err)
	}

	b, err := testBoard(testBytes)
	if err != nil {
		t.Fatalf(`Failure making test board: %v`, err)
	}

	// remove on empty store should fail
	if err = store.Remove(b); err == nil {
		t.Errorf("should error when removing non-existant board: %v", err)
	}

	if store.numBoards != 0 {
		t.Errorf("failed remove should not decrement count")
	}

	if err = store.Add(b); err != nil {
		t.Errorf("error saving over board")
	}

	if store.numBoards != 1 {
		t.Errorf("inaccurate board count")
	}

	if err = store.Remove(b); err != nil {
		t.Errorf("error removing valid board: %v", err)
	}

	if store.numBoards != 0 {
		t.Errorf("inaccurate board count")
	}
}
