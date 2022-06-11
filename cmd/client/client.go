package main

import (
	"fmt"
	"log"

	"github.com/royragsdale/s83"
)

func main() {
	testPublisher, err := s83.NewPublisherFromKey(s83.TestPublic)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("test publisher : %s\n", testPublisher)

	testCreator, err := s83.NewCreatorFromKey(s83.TestPrivate)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("test creator   : %s\n", testCreator)

	board, err := testCreator.NewBoard([]byte("foo"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(board)

	fmt.Println("generating a new creator. please be patient")
	creator, err := s83.NewCreator()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("new creator: %s\n", creator)

}
