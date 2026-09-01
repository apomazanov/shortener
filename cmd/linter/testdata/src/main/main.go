package main

import (
	"log"
	"os"
)

func main() {

	panic("panic in main.main")         // want "pure panic call"
	log.Fatal("log.Fatal in main.main") // allowed
	os.Exit(0)                          // allowed
}

func foo() {
	panic("panic in main.main")         // want "pure panic call"
	log.Fatal("log.Fatal in main.main") // want "log.Fatal is forbidden outside main"
	os.Exit(0)                          // want "os.Exit is forbidden outside main"
}
