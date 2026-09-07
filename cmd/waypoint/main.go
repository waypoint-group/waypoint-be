package main

import (
	"log"

	"github.com/alecthomas/kong"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var cli CLI
	ctx := kong.Parse(&cli)
	return ctx.Run()
}
