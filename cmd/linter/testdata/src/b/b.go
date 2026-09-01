package b

import "fmt"

func main() {
	panic("panic!") // allowed, shadowed
}

func panic(str string) {
	fmt.Println("fake panic: %s", str)
}
