package util

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

func (s Set[T]) Add(item T) error {
	if IsEmpty(item) {
		return fmt.Errorf("cannot add empty value into set")
	}
	s[item] = struct{}{}
	return nil
}

func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

func (s Set[T]) Remove(item T) {
	delete(s, item)
}

func (s Set[T]) Map(f func(T) T) Set[T] {
	result := NewSet[T]()
	for item := range s {
		_ = result.Add(f(item))
	}
	return result
}

func (s Set[T]) ValuesChan() <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for item := range s {
			ch <- item
		}
	}()
	return ch
}

func (s Set[T]) Values() []T {
	result := make([]T, 0, len(s))
	for item := range s {
		result = append(result, item)
	}
	return result
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) IsEmpty() bool {
	return len(s) == 0
}

func IsEmpty[T any](val T) bool {
	return reflect.DeepEqual(val, *new(T))
}

func (s Set[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Values()) // Serialize as a slice
}

func (s Set[T]) MarshalYAML() (interface{}, error) {
	return s.Values(), nil // Serialize as a slice
}
