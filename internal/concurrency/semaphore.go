package concurrency

type empty struct{}
type Semaphore chan empty

func NewSemaphore(capacity int) Semaphore {
	return make(Semaphore, capacity)
}

// Acquire reserves n amount of resources from the Semaphore.
func (s Semaphore) Acquire(n int) {
	e := empty{}
	for i := 0; i < n; i++ {
		s <- e
	}
}

// Release releases n amount of resources from the Semaphore.
func (s Semaphore) Release(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

// Lock acquires a single resource from the Semaphore.
func (s Semaphore) Lock() {
	s.Acquire(1)
}

// Unlock waits to release a single resource from the Semaphore.
func (s Semaphore) Unlock() {
	s.Release(1)
}
