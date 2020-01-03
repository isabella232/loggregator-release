package dns

import (
	"math/rand"
	"sync"
	"time"
)

var (
	r  = rand.New(rand.NewSource(time.Now().UnixNano()))
	mu sync.Mutex
)

// Intn implements rand.Intn on the grpcrand global source.
func Intn(n int) int {
	mu.Lock()
	res := r.Intn(n)
	mu.Unlock()
	return res
}

// Float64 implements rand.Float64 on the grpcrand global source.
func Float64() float64 {
	mu.Lock()
	res := r.Float64()
	mu.Unlock()
	return res
}
