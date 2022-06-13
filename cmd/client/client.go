package main

import (
	"encoding/hex"
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
	fmt.Println("new creator :", creator)
	fmt.Println("private     :", hex.EncodeToString(creator.PrivateKey))

	board, err := creator.NewBoard([]byte("hello world"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(board)

}
