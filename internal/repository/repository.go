package repository

import "fmt"

var storage = make(map[string]string)
var counter int

func AddRecord(long string) (string, bool) {
	counter++
	short := fmt.Sprintf("short%d", counter)
	// fmt.Printf("Adding value: %s, new key is: %s\r\n", long, short)
	storage[short] = long
	return short, true
}

func GetRecord(short string) (string, bool) {
	long, ok := storage[short]
	// fmt.Printf("Searching key: %s, returning: %s\r\n", short, long)
	return long, ok
}
