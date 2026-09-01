package resetter

import (
	"fmt"
	"reflect"
	"sync"
)

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	objects []T
	mu      sync.Mutex
	create  func() T
}

func New[T Resetter](creator func() T) (*Pool[T], error) {

	if creator == nil {
		return nil, fmt.Errorf("creator must not be nil")
	}

	return &Pool[T]{create: creator}, nil
}

func (p *Pool[T]) Get() T {
	p.mu.Lock()

	sz := len(p.objects)
	var zero T

	if sz == 0 {
		p.mu.Unlock()
		return p.create()
	}

	lastIdx := sz - 1
	obj := p.objects[lastIdx]
	p.objects[lastIdx] = zero
	p.objects = p.objects[:lastIdx]

	p.mu.Unlock()
	return obj

}

func (p *Pool[T]) Put(obj T) {

	if reflect.ValueOf(obj).IsNil() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.objects = append(p.objects, obj)
}
