package repository

import (
	"fmt"
)

type InMemoryRepository struct {
	data  map[string]string
	counter int
}

/* -------------------------------------------------------------------------- */
func NewInMemoryRepo() *InMemoryRepository {
	return &InMemoryRepository{data: make(map[string]string)}
}

/* -------------------------------------------------------------------------- */
func (r *InMemoryRepository) Add(long string) (string, bool) {
	r.counter++
	short := fmt.Sprintf("short%d", r.counter)
	r.data[short] = long
	return short, true
}

/* -------------------------------------------------------------------------- */
func (r *InMemoryRepository) Get(short string) (string, bool) {
	long, ok := r.data[short]
	return long, ok
}
