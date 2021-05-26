package dproxy

import (
	"log"
	"os"
)

// global logger
var L = log.New(os.Stdout, "dproxy: ", log.Lshortfile|log.LstdFlags)
