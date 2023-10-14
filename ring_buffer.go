package main

import (
	"errors"
)

type RingBuffer[T any] struct {
	data []T
	size int
	head int
	tail int
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, size+1),
		size: size + 1,
		head: 0,
		tail: 0,
	}
}

func (rb *RingBuffer[T]) Enqueue(value T) error {
	next := (rb.tail + 1) % rb.size
	if next == rb.head {
		return errors.New("ring buffer is full")
	}
	rb.data[rb.tail] = value
	rb.tail = next
	return nil
}

func (rb *RingBuffer[T]) Dequeue() (T, error) {
	if rb.IsEmpty() {
		var zero T
		return zero, errors.New("ring buffer is empty")
	}
	value := rb.data[rb.head]
	rb.head = (rb.head + 1) % rb.size
	return value, nil
}

func (rb *RingBuffer[T]) IsEmpty() bool {
	return rb.head == rb.tail
}
