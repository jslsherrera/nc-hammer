package main

import (
	"log"

	"github.com/damianoneill/nc-hammer/cmd"
)

var (
	// VERSION should be set as an argument in the build
	VERSION = "NOT SET"
)

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	cmd.Execute(VERSION)
}
