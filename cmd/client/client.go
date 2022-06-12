package main

import (
	"fmt"
	"log"

	"github.com/royragsdale/s83"
)

func main() {

	fmt.Println("generating a new creator. please be patient")
	creator, err := s83.NewCreator()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("new creator: %s\n", creator)

}
