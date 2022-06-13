package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/royragsdale/s83"
)

func main() {

	config := loadConfig()
	dispatchCommand(config)

	/*
		board, err := creator.NewBoard([]byte("hello world"))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(board)
	*/

}

// ref: https://gobyexample.com/command-line-subcommands
func dispatchCommand(config Config) {

	// TODO: add global flags (e.g. config/verbose)

	// New creator
	// TODO: add flags for, difficulty, check existence
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)

	// Publish a board
	// TODO: add flags to store locally, board on CLI, board from file
	pubCmd := flag.NewFlagSet("pub", flag.ExitOnError)

	// Get boards from a server
	// TODO: add flags to store, launch browser
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("expected 'get', 'pub', or 'new' subcommand")
		os.Exit(1)
	}

	// Check which subcommand is invoked.
	switch os.Args[1] {

	// For every subcommand, we parse its own flags and
	// have access to trailing positional arguments.
	case "new":
		newCmd.Parse(os.Args[2:])
		New(config, newCmd)
	case "pub":
		pubCmd.Parse(os.Args[2:])
		fmt.Println("TODO: subcommand 'pub' not yet implemented")
	case "get":
		getCmd.Parse(os.Args[2:])
		fmt.Println("TODO: subcommand 'get' not yet implemented")

	default:
		fmt.Println("expected 'get', 'pub', or 'new' subcommand")
		os.Exit(1)
	}
}

func New(config Config, args *flag.FlagSet) {
	fmt.Println("[info] Generating a new creator key. Please be patient.")
	start := time.Now()
	creator, cnt, err := s83.NewCreator()
	if err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	elapsed := t.Sub(start).Seconds()
	kps := int(float64(cnt) / elapsed)

	fmt.Printf("[info] Success! Found a valid key in %d iterations over %d seconds (%d kps)\n", cnt, int(elapsed), kps)
	fmt.Println("[info] The public key is your creator id. Share it!")
	fmt.Println("[WARN] The secret key is SECRET. Do not share it or lose it.")
	fmt.Println("public:", creator)
	fmt.Println("secret:", creator.ExportPrivateKey())
}
