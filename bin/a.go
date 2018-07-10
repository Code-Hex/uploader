package main

import (
	"fmt"
	"log"
)

func main() {
	if true {
		log.Panic("Hello")
	}
	fmt.Println("World")
}
