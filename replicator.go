package main

import (
	"context"
	"sync"
	"time"
)

type replicator[T any] struct {
	sync.Mutex
	consumers []chan T
	c         chan T
}

func newReplicator[T any](ctx context.Context) *replicator[T] {
	r := &replicator[T]{
		consumers: make([]chan T, 0),
		c:         make(chan T),
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case item := <-r.c:
				for _, consumer := range r.consumers {
					select {
					case consumer <- item:
					case <-time.After(10 * time.Millisecond):
						continue
					}
				}
			}
		}
	}()

	return r
}

func (r *replicator[T]) consume() chan T {
	consumer := make(chan T)

	defer lockUnlock(r)
	r.consumers = append(r.consumers, consumer)

	return consumer
}

func (r *replicator[T]) produce(item T) chan struct{} {
	wait := make(chan struct{})
	go func() {
		defer close(wait)
		r.c <- item
	}()

	return wait
}
