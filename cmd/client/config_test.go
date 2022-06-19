package main

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {

	dir := t.TempDir()

	err := os.Setenv("XDG_CONFIG_HOME", dir)
	if err != nil {
		t.Fatalf(`Failure setting "XDG_CONFIG_HOME": %v`, err)
	}

	loadConfig(defaultConfigName)
}
