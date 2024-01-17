package internal

import (
	"log"
	"os"
)

// Default ORM logger
var Logger *log.Logger

func init() {
	Logger = log.New(os.Stderr, "(ORM) ", log.Ltime)
}
