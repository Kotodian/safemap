package safemap

import "sync"

const (
	copyThreshold = 1000
	maxDeletion   = 10000
)

// SafeMap provides map alternative to avoid memory leak.
// This implementation is not needed until issue below fixed.
// https://github.com/golang/go/issues/20135
type SafeMap[K comparable, V any] struct {
	lock        sync.RWMutex
	deletionOld int
	deletionNew int
	dirtyOld    map[K]V
	dirtyNew    map[K]V
}

// NewSafeMap returns a SafeMap.
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{dirtyOld: make(map[K]V), dirtyNew: make(map[K]V)}
}

// Get gets the value with the given key from m.
func (m *SafeMap[K, V]) Get(key K) (V, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if val, ok := m.dirtyOld[key]; ok {
		return val, true
	}

	val, ok := m.dirtyNew[key]
	return val, ok
}

func (m *SafeMap[K, V]) Size() int {
	m.lock.RLock()
	size := len(m.dirtyOld) + len(m.dirtyNew)
	m.lock.RUnlock()
	return size
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *SafeMap[K, V]) Range(f func(key K, val V) bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for k, v := range m.dirtyOld {
		if !f(k, v) {
			return
		}
	}

	for k, v := range m.dirtyNew {
		if !f(k, v) {
			return
		}
	}
}

func (m *SafeMap[K, V]) Set(key K, value V) {
	m.lock.Lock()
	if m.deletionOld <= maxDeletion {
		if _, ok := m.dirtyNew[key]; ok {
			delete(m.dirtyNew, key)
			m.deletionNew++
		}
		m.dirtyOld[key] = value
	} else {
		if _, ok := m.dirtyOld[key]; ok {
			delete(m.dirtyOld, key)
			m.deletionOld++
		}
		m.dirtyNew[key] = value
	}
	m.lock.Unlock()
}

// Del deletes the value with the given key from m.
func (m *SafeMap[K, V]) Del(key K) {
	m.lock.Lock()
	if _, ok := m.dirtyOld[key]; ok {
		delete(m.dirtyOld, key)
		m.deletionOld++
	} else if _, ok := m.dirtyNew[key]; ok {
		delete(m.dirtyNew, key)
		m.deletionNew++
	}
	if m.deletionOld > maxDeletion && len(m.dirtyOld) < copyThreshold {
		for k, v := range m.dirtyOld {
			m.dirtyNew[k] = v
		}
		m.dirtyOld = m.dirtyNew
		m.deletionOld = m.deletionNew
		m.dirtyNew = make(map[K]V)
		m.deletionNew = 0
	}
	if m.deletionNew > maxDeletion && len(m.dirtyNew) < copyThreshold {
		for k, v := range m.dirtyNew {
			m.dirtyOld[k] = v
		}
		m.dirtyNew = make(map[K]V)
		m.deletionNew = 0
	}
	m.lock.Unlock()
}
