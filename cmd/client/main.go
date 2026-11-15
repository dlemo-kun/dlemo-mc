package main

import (
	"log"

	"github.com/dlemo-kun/dlemo-mc/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}