package main

import (
	"log"

	"github.com/SShogun/redisforge/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
