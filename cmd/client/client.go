package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/royragsdale/s83"
)

// ref: https://gobyexample.com/command-line-subcommands
func main() {

	// seed for nonces
	rand.Seed(time.Now().Unix())

	// TODO: add global verbose flag
	var confFlag = flag.String("c", defaultConfigName, "name of profile file to use")

	// New creator
	// TODO: add flags to save/export as a config
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	jFlag := newCmd.Int("j", 1, "number of miners to run concurrently")

	// Display configuration information (e.g. which "profile") is in use
	whoCmd := flag.NewFlagSet("who", flag.ExitOnError)

	// Publish a board
	// TODO: handle "delete" functionality, aka "tombstone" boards, "404 Not Found"
	pubCmd := flag.NewFlagSet("pub", flag.ExitOnError)
	dryFlag := pubCmd.Bool("dry", false, "dry run, print board locally instead of publishing")

	// Get boards from a server
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	outFlag := getCmd.String("o", "", "output your 'Daily Spring' to a specific path")
	browseFlag := getCmd.Bool("go", false, "open your 'Daily Spring' in a browser")
	newOnlyFlag := getCmd.Bool("new", false, "only get new boards")

	cmdOrder := []string{"pub", "get", "new", "who"}
	cmds := map[string]struct {
		fs          *flag.FlagSet
		description string
	}{
		"pub": {pubCmd, "publish a board"},
		"get": {getCmd, "download follows/boards and make your 'Daily Spring'"},
		"new": {newCmd, "generate a new keypair"},
		"who": {whoCmd, "show profile information"},
	}

	flag.Usage = func() {
		fmt.Println("usage: s83 [flags] <command>")

		fmt.Println("\ncommands:")
		for _, cName := range cmdOrder {
			cmd := cmds[cName]
			fmt.Printf("  %-8s %s\n", cName, cmd.description)

		}

		fmt.Println("\nflags:")
		flag.PrintDefaults()

		fmt.Println("\nUse \"s83 <command> -h\" for more information about a command.")
	}

	pubCmd.Usage = func() {
		fmt.Printf("%s: %s\n", "pub", cmds["pub"].description)
		fmt.Println("\nusage: s83 pub [flags] <path>")

		fmt.Println("\nflags:")
		pubCmd.PrintDefaults()
	}

	getCmd.Usage = func() {
		fmt.Printf("%s: %s\n", "get", cmds["get"].description)
		fmt.Println("\nusage: s83 get [flags] [key]")

		fmt.Println("\nflags:")
		getCmd.PrintDefaults()

		fmt.Println("\noptional:")
		fmt.Printf("  %-8s %s\n", "key", "get a single board")
	}

	newCmd.Usage = func() {
		fmt.Printf("%s: %s\n", "new", cmds["new"].description)
		fmt.Println("\nusage: s83 new [flags]")
		fmt.Println("\nflags:")
		newCmd.PrintDefaults()
	}

	whoCmd.Usage = func() {
		fmt.Printf("%s: %s\n", "who", cmds["who"].description)
		fmt.Println("\nusage: s83 who")
	}

	// parse global flags
	flag.Parse()
	config := loadConfig(*confFlag)

	if flag.NArg() < 1 {
		flag.Usage()
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
			fmt.Println("<path> to file to be published is required")
			pubCmd.Usage()
			os.Exit(1)
		}

		if !config.Creator.Valid() {
			fmt.Println("[ERROR] Invalid creator configuration.")
			fmt.Println("[info] use `s83 new` to a 'secret'")
			fmt.Printf("[info] then add a 'secret=' line to your config file (%s)\n", config.Path())
			os.Exit(1)
		}

		if config.Server == nil && !*dryFlag {
			fmt.Println("[ERROR] missing server configuration.")
			fmt.Printf("[info] add a 'server=' line to your config file (%s)\n", config.Path())
			os.Exit(1)
		}

		config.Pub(pubCmd.Arg(0), *dryFlag)

	case "get":
		getCmd.Parse(subArgs)
		if getCmd.NArg() > 1 {
			getCmd.Usage()
			os.Exit(1)
		}

		config.Get(getCmd.Arg(0), *outFlag, *browseFlag, *newOnlyFlag)

	default:
		fmt.Printf("invalid command\n\n")
		flag.Usage()
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

func (config Config) Pub(path string, dryRun bool) {
	data, err := os.ReadFile(path)
	exitOnError(err)

	board, err := config.Creator.NewBoard(data)
	exitOnError(err)

	if !dryRun {
		exitOnError(publishBoard(config.Server, board))
	} else {
		fmt.Println("[info] Success. This board should publish (pending TTL checks)")
		fmt.Println("[info] Size: ", len(board.Content))
		fmt.Println(board)
	}
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

// TODO: Clients should scan for the <link rel="next"> element:
// <link rel="next" href="<URL>">

// TODO: the client may also scan for arbitrary data stored in
// data-spring-* attributes throughout the board.

func (config Config) Get(key string, outPath string, browse bool, newOnly bool) {
	follows := config.Follows

	// single key specified
	if key != "" {
		keyURL := config.Server
		keyURL.Path = path.Join(keyURL.Path, key)
		f, err := s83.NewFollow(key, keyURL.String(), "")
		exitOnError(err)
		follows = []s83.Follow{f}
	}

	errCnt := 0
	newBoards := map[string]s83.Board{}
	localBoards := map[string]s83.Board{}
	for _, f := range follows {
		key := f.Key()

		// default to omit modified time header
		modTimeStr := ""

		// local local copy (if exists)
		localBoard, err := s83.BoardFromPath(config.followToPath(f))
		if err == nil {
			// get our copy of the timestamp (only want boards newer than this)
			modTimeStr = localBoard.Timestamp()
			localBoards[key] = localBoard
		}

		// fetch board from server
		b, err := f.GetBoard(modTimeStr)
		if err != nil {
			// TODO: improve error handling, actual checks not string inference
			if strings.Contains(err.Error(), "304 Not Modified") {
				fmt.Printf("[info] 304 - no new board for %s\n", f)
			} else {
				fmt.Printf("[warn] failed to get board for %s: %v\n", f, err)
				errCnt += 1
			}
			continue
		}

		// some servers don't reply Not Modified, making all boards seem new
		// do a local check to allow follow on styling/alerting
		if b.SameAs(localBoard) {
			continue
		}

		// Actually got a new board. Save to disk.
		b.Save(config.DataPath())
		newBoards[key] = b
	}

	if newOnly {
		if len(newBoards) == 0 {
			fmt.Println("[info] no new boards")
			os.Exit(0)
		}
		// zero out pre-existing local boards
		localBoards = map[string]s83.Board{}
	}

	// if output was not set via a command line flag store at default location
	if outPath == "" {
		outPath = config.outPath()
	}

	outF, err := os.Create(outPath)
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}
	defer outF.Close()

	// actually write 'The Daily' out to a file
	config.renderBoards(newBoards, localBoards, outF)

	// output results message
	if errCnt == 0 {
		fmt.Printf("[info] Success. Saved %d boards to %s\n", len(newBoards), config.DataPath())
	} else if errCnt == len(follows) {
		exitOnError(errors.New("Failed to get any boards"))
	} else {
		fmt.Printf("[warn] Failed to get %d/%d boards\n", errCnt, len(follows))
	}

	fmt.Printf("[info] Published your 'Daily Spring' to: %s\n", outPath)

	if browse {
		err := openBrowserToPath(outPath)
		if err != nil {
			fmt.Printf("[warn] Failed launching a browser: %v\n", err)
		} else {
			fmt.Println("[info] Success. check your browser for your 'Daily Spring'")
		}
	}
}

func openBrowserToPath(path string) error {
	// TODO: generalize for other launchers/platform, and better error checking
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}

func (config Config) followToPath(f s83.Follow) string {
	return filepath.Join(config.DataPath(), fmt.Sprintf("%s.s83", f.Key()))
}

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}
}
