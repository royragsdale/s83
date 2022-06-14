package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/royragsdale/s83"
)

func main() {
	config := loadConfig()
	dispatchCommand(config)
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
	pubCmd.Usage = func() {
		fmt.Fprintf(pubCmd.Output(), "usage: pub <path>\n")
	}

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
		if pubCmd.NArg() != 1 {
			pubCmd.Usage()
			fmt.Println("<path> to file to be published is required")
			os.Exit(1)
		}

		if !config.Creator.Valid() {
			fmt.Println("[ERROR] Invalid creator configutation.")
			fmt.Println("[info] use `s83 new` to a 'secret'")
			fmt.Printf("[info] then add a 'secret=' line to your config file (%s)\n", configPath())
			os.Exit(1)
		}

		if config.Server == nil {
			fmt.Println("[ERROR] missing server configutation.")
			fmt.Printf("[info] add a 'server=' line to your config file (%s)\n", configPath())
			os.Exit(1)
		}
		Pub(config, pubCmd.Arg(0))

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

	// actually generate the new creator
	creator, cnt, err := s83.NewCreator()
	if err != nil {
		log.Fatal(err)
	}
	// compute mildly interesting stats
	t := time.Now()
	elapsed := t.Sub(start).Seconds()
	kps := int(float64(cnt) / elapsed)

	// display results
	fmt.Printf("[info] Success! Found a valid key in %d iterations over %d seconds (%d kps)\n", cnt, int(elapsed), kps)
	fmt.Println("[info] The public key is your creator id. Share it!")
	fmt.Println("[WARN] The secret key is SECRET. Do not share it or lose it.")
	fmt.Println("public:", creator)
	fmt.Println("secret:", creator.ExportPrivateKey())
}

func Pub(config Config, path string) {
	data, err := os.ReadFile(path)
	exitOnError(err)

	board, err := config.Creator.NewBoard(data)
	exitOnError(err)

	exitOnError(publishBoard(config.Server, board))
}

func publishBoard(server *url.URL, board s83.Board) error {

	// add publisher key
	server.Path = path.Join(server.Path, board.Publisher.String())
	fmt.Println(server.String())

	client := &http.Client{}
	req, err := http.NewRequest("PUT", server.String(), bytes.NewReader(board.Content))
	exitOnError(err)

	// set headers
	req.Header.Set("Spring-Version", s83.SpringVersion)
	req.Header.Set("Authorization", fmt.Sprintf("Spring-83 Signature=%s", board.Signature()))
	req.Header.Set("If-Unmodified-Since", board.Timestamp())

	// make request
	res, err := client.Do(req)
	exitOnError(err)

	// read response
	body, err := io.ReadAll(res.Body)
	exitOnError(err)

	if res.StatusCode == http.StatusOK {
		fmt.Println("[info] Success")
		return nil
	} else {
		msg := fmt.Sprintf("%s: %s", res.Status, body)
		return errors.New(msg)
	}
}

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}
}
