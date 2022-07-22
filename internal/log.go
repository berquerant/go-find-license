package internal

import (
	"fmt"
	"log"
)

var debug bool

// EnableDebug enables debug logs.
func EnableDebug() { debug = true }

// Debugf writes debug logs.
func Debugf(format string, v ...any) {
	if debug {
		log.Printf("[DEBUG] %s\n", fmt.Sprintf(format, v...))
	}
}

// Infof writes info logs.
func Infof(format string, v ...any) {
	log.Printf("[INFO] %s\n", fmt.Sprintf(format, v...))
}
