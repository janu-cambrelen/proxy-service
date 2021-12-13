package main

import (
	"log"

	"github.com/janu-cambrelen/proxy-service/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
