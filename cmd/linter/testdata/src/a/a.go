package a

import (
	"log"
	"os"
)

func main() {

	panic("panic in a.main")         // want "pure panic call"
	log.Fatal("log.Fatal in a.main") // want "log.Fatal is forbidden outside main"
	os.Exit(0)                       // want "os.Exit is forbidden outside main"
}

func foo() {
	panic("panic in a.foo")         // want "pure panic call"
	log.Fatal("log.Fatal in a.foo") // want "log.Fatal is forbidden outside main"
	os.Exit(0)                      // want "os.Exit is forbidden outside main"
}
