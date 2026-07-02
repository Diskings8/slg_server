package hashmaps

import "github.com/go4org/hashtriemap"

type Map[K comparable, V any] struct {
	hashtriemap.HashTrieMap[K, V]
}

// Len returns the number of elements in the map.
func (m *Map[K, V]) Len() int {
	var count int
	m.Range(func(K, V) bool {
		count++
		return true
	})
	return count
}

// Keys returns all keys in the map.
func (m *Map[K, V]) Keys() []K {
	var keys []K
	m.Range(func(key K, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}
