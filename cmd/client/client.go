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

// ref: https://gobyexample.com/command-line-subcommands
func main() {

	// TODO: add global verbose flag
	var confFlag = flag.String("c", defaultConfigName, "name of configuration file to use")

	// New creator
	// TODO: add flags for, difficulty, check existence
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	var jFlag = newCmd.Int("j", 1, "number of miners to run concurrently")

	// Display configuration information (e.g. which "profile") is in use
	whoCmd := flag.NewFlagSet("who", flag.ExitOnError)

	// Publish a board
	// TODO: add flags to store locally, board on CLI, board from file
	pubCmd := flag.NewFlagSet("pub", flag.ExitOnError)
	pubCmd.Usage = func() {
		fmt.Fprintf(pubCmd.Output(), "usage: pub <path>\n")
	}

	// Get boards from a server
	// TODO: add flags to store, launch browser, set mod time (e.g. from local copy)
	// TODO: saved list of boards to fetch (e.g. subscription)
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getCmd.Usage = func() {
		fmt.Fprintf(getCmd.Output(), "usage: get [public]\n")
	}

	cmds := []struct {
		name string
		fs   *flag.FlagSet
	}{
		{"pub", pubCmd},
		{"get", getCmd},
		{"new", newCmd},
		{"who", whoCmd},
	}

	// TODO: list of commands with descriptions
	// build up expected usage
	expectedStr := "expected a subcommand:"
	for i, cmd := range cmds {
		if i > 0 {
			expectedStr += ","
		}
		if i == len(cmds)-1 {
			expectedStr += " or"
		}
		expectedStr += fmt.Sprintf(` '%s'`, cmd.name)
	}

	// parse global flags
	flag.Parse()
	config := loadConfig(*confFlag)

	if flag.NArg() < 1 {
		// TODO: proper usage
		fmt.Println(expectedStr)
		os.Exit(1)
	}
	subArgs := flag.Args()[1:]

	// Check which subcommand is invoked.
	switch flag.Arg(0) {

	case "new":
		newCmd.Parse(subArgs)
		config.New(*jFlag)

	case "who":
		whoCmd.Parse(subArgs)
		config.Who()

	case "pub":
		pubCmd.Parse(subArgs)
		if pubCmd.NArg() != 1 {
			pubCmd.Usage()
			fmt.Println("<path> to file to be published is required")
			os.Exit(1)
		}

		if !config.Creator.Valid() {
			fmt.Println("[ERROR] Invalid creator configuration.")
			fmt.Println("[info] use `s83 new` to a 'secret'")
			fmt.Printf("[info] then add a 'secret=' line to your config file (%s)\n", config.Path())
			os.Exit(1)
		}

		if config.Server == nil {
			fmt.Println("[ERROR] missing server configuration.")
			fmt.Printf("[info] add a 'server=' line to your config file (%s)\n", config.Path())
			os.Exit(1)
		}

		config.Pub(pubCmd.Arg(0))

	case "get":
		getCmd.Parse(subArgs)
		if getCmd.NArg() > 1 {
			getCmd.Usage()
			os.Exit(1)
		}

		config.Get(getCmd.Arg(0))

	default:
		fmt.Println(expectedStr)
		os.Exit(1)
	}
}

func (config Config) New(j int) {
	fmt.Printf("[info] Generating a new creator key with %d miners. Please be patient.\n", j)
	start := time.Now()

	// actually generate the new creator
	c := s83.NewCreator(j)
	if c.Err != nil {
		log.Fatal(c.Err)
	}
	// compute mildly interesting stats
	t := time.Now()
	elapsed := t.Sub(start).Seconds()
	kps := int(float64(c.Count) / elapsed)

	// display results
	fmt.Printf("[info] Success! Found a valid key in %d iterations over %d seconds (%d kps)\n", c.Count, int(elapsed), kps)
	fmt.Println("[info] The public key is your creator id. Share it!")
	fmt.Println("[WARN] The secret key is SECRET. Do not share it or lose it.")
	fmt.Println("public:", c.Creator)
	fmt.Println("secret:", c.Creator.ExportPrivateKey())
}

func (config Config) Who() {
	fmt.Print(config)
}

func (config Config) Pub(path string) {
	data, err := os.ReadFile(path)
	exitOnError(err)

	board, err := config.Creator.NewBoard(data)
	exitOnError(err)

	exitOnError(publishBoard(config.Server, board))
}

func publishBoard(server *url.URL, board s83.Board) error {

	// add publisher key to URL
	server.Path = path.Join(server.Path, board.Publisher.String())

	client := &http.Client{}
	req, err := http.NewRequest("PUT", server.String(), bytes.NewReader(board.Content))
	exitOnError(err)

	// set headers
	req.Header.Set("Content-Type", "text/html;charset=utf-8")
	req.Header.Set("Spring-Version", s83.SpringVersion)
	req.Header.Set("Spring-Signature", board.Signature())

	// TODO(?): If-Unmodified-Since: <date and time in UTC, HTTP (RFC 5322) format>
	req.Header.Set("If-Unmodified-Since", board.Timestamp())

	// make request
	res, err := client.Do(req)
	exitOnError(err)

	// read response
	body, err := io.ReadAll(res.Body)
	exitOnError(err)

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		fmt.Println("[info] Success")
		return nil
	} else {
		msg := fmt.Sprintf("%s: %s", res.Status, body)
		return errors.New(msg)
	}
}

// TODO: realm/trust management
// "If the signature is not valid,the client must drop the response and
// remove the server from its list of trustworthy peers

// TODO: situate each board inside its own Shadow DOM (combine multiple boards?)

// TODO: Clients should scan for the <link rel="next"> element:
// <link rel="next" href="<URL>">

// TODO: the client may also scan for arbitrary data stored in
// data-spring-* attributes throughout the board.

// TODO: Content Security Policy (CSP) to prevent images and js/fonts/media

// TODO: display each board in a region with an aspect ratio of either 1:sqrt(2) or sqrt(2):1

// cli only at the moment > to a file and view in a browser
func (config Config) Get(key string) {
	follows := config.Follows

	// single key specified
	if key != "" {
		keyURL := config.Server
		keyURL.Path = path.Join(keyURL.Path, key)
		f, err := s83.NewFollow(key, keyURL.String(), "")
		exitOnError(err)
		follows = []s83.Follow{f}
	}

	boards := []s83.Board{}
	for _, f := range follows {
		b, err := f.GetBoard()
		if err != nil {
			fmt.Printf("[warn] failed to get board for %s: %v\n", f, err)
			continue
		}
		boards = append(boards, b)
	}
	for _, b := range boards {
		b.Save(config.DataPath())
	}
	fmt.Printf("[info] Success. Saved %d boards to %s\n", len(boards), config.DataPath())
}

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}
}
