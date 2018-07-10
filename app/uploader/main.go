package main

import (
	"os"

	"github.com/Code-Hex/upload/server/service/uploader"
)

func main() {
	os.Exit(uploader.NewService().Run())
}
