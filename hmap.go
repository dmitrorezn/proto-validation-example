package main

import "sync"

func newHmap[K comparable, V any](size int) hmap[K, V] {
	return make(hmap[K, V], size)
}

func lockUnlock(locker sync.Locker) func() {
	locker.Lock()

	return locker.Unlock
}

func newSafe[K comparable, V any]() *safehmap[K, V] {
	return &safehmap[K, V]{
		hmap: make(hmap[K, V]),
	}
}

type safehmap[K comparable, V any] struct {
	sync.RWMutex
	hmap hmap[K, V]
}

func (m *safehmap[K, V]) Put(key K, val V) {
	defer lockUnlock(m)()

	m.Put(key, val)
}
func (m *safehmap[K, V]) Del(key K) {
	defer lockUnlock(m)()

	m.Del(key)
}
func (m *safehmap[K, V]) Get(key K) (V, bool) {
	defer lockUnlock(m.RLocker())()
	v, ok := m.Get(key)

	return v, ok
}

type hmap[K comparable, V any] map[K]V

func (m hmap[K, V]) Put(key K, val V) {
	m[key] = val
}
func (m hmap[K, V]) Del(key K) {
	delete(m, key)
}
func (m hmap[K, V]) Get(key K) (V, bool) {
	v, ok := m[key]

	return v, ok
}
