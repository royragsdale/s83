package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStore(t *testing.T) {

	dir := t.TempDir()

	store, err := loadStore(dir)
	if err != nil {
		t.Errorf(`An empty directory store should be valid: %v`, err)
	}

	if store.NumBoards != 0 {
		t.Errorf("An empty directory should have 0 boards")
	}

	fPath := filepath.Join(dir, "f")
	err = os.WriteFile(fPath, []byte("file"), 0644)
	if err != nil {
		t.Errorf(`Failed setting up test file: %v`, err)
	}

	_, err = loadStore(fPath)
	if err == nil {
		t.Errorf("A file is not a valid store, should error")
	}

	// TODO: directories with valid and invalid boards

}
