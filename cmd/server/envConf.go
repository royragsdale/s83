package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"

	"github.com/royragsdale/s83"
)

const envHost = "HOST"
const envPort = "PORT"
const envStore = "STORE"
const envTTL = "TTL"
const envTitle = "TITLE"
const envAdmin = "ADMIN_BOARD"

var envVars = []string{envHost, envPort, envStore, envTTL, envTitle, envAdmin}

var defaultVars = map[string]string{
	envHost:  "",
	envPort:  "8080",
	envStore: "store",
	envTTL:   "22",
	envTitle: "s83d",
	envAdmin: "",
}

type Server struct {
	host        string
	port        int
	store       *Store
	ttl         int // days
	title       string
	admin       *s83.Publisher
	blockList   map[string]bool
	testCreator s83.Creator // test key
	templates   *template.Template
}

func NewServerFromEnv() *Server {

	// configurable from environment variables
	host := varOrDefault(envHost)
	port := intOrDefault(envPort)
	ttl := intOrDefault(envTTL)
	storePath := varOrDefault(envStore)
	title := varOrDefault(envTitle)
	adminKey := varOrDefault(envAdmin)

	if ttl < 7 || ttl > 22 {
		log.Fatalf("Invalid TTL (%d), must not be less than 7 or more than 22 days.", ttl)
	}

	// TODO: add server private key
	// TODO: load block list from a board
	// used for both GET and PUT
	blockList := map[string]bool{
		s83.InfernalKey: true,
	}

	// creator for the test key board
	testCreator, err := s83.NewCreatorFromKey(s83.TestPrivate)
	if err != nil {
		log.Fatal(err)
	}

	// admin board
	var admin *s83.Publisher = nil
	adminPub, err := s83.NewPublisherFromKey(adminKey)
	if err != nil {
		log.Println("no admin board configured")
	} else {
		admin = &adminPub
		log.Println("admin board configured for ", adminPub)
	}

	// pre load store
	store, err := loadStore(storePath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d boards from store %s", store.NumBoards, store.Dir)

	// load templates
	templates := template.Must(template.ParseFS(resources, "templates/*.tmpl"))

	srv := &Server{
		host,
		port,
		store,
		ttl,
		title,
		admin,
		blockList,
		testCreator,
		templates,
	}

	log.Printf("board TTL: %d (days)", srv.ttl)
	return srv
}

func envUsage() {
	fmt.Println("Usage: s83d is designed to be configured using environment variables.")
	fmt.Printf("\nFor example: `PORT=8383 ./s83d`\n\n")
	fmt.Printf("%-16s %s\n", "variable", "default")
	fmt.Printf("%-16s %s\n", "--------", "-------")
	for _, name := range envVars {
		fmt.Printf("%-16s %v\n", name, defaultVars[name])
	}

}

func varOrDefault(name string) string {
	envVal := os.Getenv(name)
	if envVal != "" {
		return envVal
	}

	// get default
	val, ok := defaultVars[name]
	if !ok {
		log.Fatalf("Attempted to get variable with no default from ENV: %s", name)
	}
	return val
}

func intOrDefault(name string) int {
	nStr := varOrDefault(name)
	n, err := strconv.Atoi(nStr)
	if err != nil {
		log.Fatalf("failed parsing int for var: %s: %s: %v\n", name, nStr, err)
	}
	return n
}

func floatOrDefault(name string) float64 {
	fStr := varOrDefault(name)
	f, err := strconv.ParseFloat(fStr, 64)
	if err != nil {
		log.Fatalf("failed parsing float for var: %s: %s: %v\n", name, fStr, err)
	}
	return f
}
