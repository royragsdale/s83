package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/royragsdale/s83"
)

const envHost = "HOST"
const envPort = "PORT"
const envStore = "STORE"
const envTTL = "TTL"
const envFactor = "DIFFICULTY"

var envVars = []string{envHost, envPort, envStore, envTTL, envFactor}

var defaultVars = map[string]string{
	envHost:   "",
	envPort:   "8080",
	envStore:  "store",
	envTTL:    "22",
	envFactor: "0.0",
}

type Server struct {
	host                string
	port                int
	store               *Store
	difficultyFactor    float64
	difficultyThreshold uint64 // TODO: simplify into Difficulty type to keep in sync
	ttl                 int    // days
	blockList           map[string]bool
	creator             s83.Creator
}

func NewServerFromEnv() *Server {

	// configurable from environment variables
	host := varOrDefault(envHost)
	port := intOrDefault(envPort)
	ttl := intOrDefault(envTTL)
	difficultyFactor := floatOrDefault(envFactor)
	storePath := varOrDefault(envStore)

	// TODO: add server private key
	// TODO: load block list from a board
	blockList := map[string]bool{s83.TestPublic: true}

	// creator for the test key board
	creator, err := s83.NewCreatorFromKey(s83.TestPrivate)
	if err != nil {
		log.Fatal(err)
	}

	// pre compute difficultyThreshold
	threshold, err := s83.DifficultyThreshold(difficultyFactor)
	if err != nil {
		log.Fatal(err)
	}

	// pre load store
	store, err := loadStore(storePath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d boards from store %s", store.NumBoards, store.Dir)

	srv := &Server{host, port, store, difficultyFactor, threshold, ttl, blockList, creator}
	log.Printf("difficulty factor: %f", srv.difficultyFactor)
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
