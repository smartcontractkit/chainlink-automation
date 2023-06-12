package ocr2keepers

import "fmt"

type ResultStore[T any] interface {
	Add(T)
	Remove(T)
	View() ([]T, error)
}

type resultStore[T any] struct {
	data map[string]T
}

func NewResultStore[T any]() *resultStore[T] {
	return &resultStore[T]{
		data: make(map[string]T),
	}
}

func (s *resultStore[T]) Add(results ...T) {
	for _, result := range results {
		key := fmt.Sprintf("%v", result)
		s.data[key] = result
	}
}

func (s *resultStore[T]) Remove(results ...T) {
	for _, result := range results {
		key := fmt.Sprintf("%v", result)
		delete(s.data, key)
	}
}

func (s *resultStore[T]) View() ([]T, error) {
	var result []T
	for _, r := range s.data {
		result = append(result, r)
	}
	return result, nil
}
