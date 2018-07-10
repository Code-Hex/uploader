package main

import (
	"os"

	"github.com/Code-Hex/upload/client"
)

func main() {
	os.Exit(client.New().Run())
}
