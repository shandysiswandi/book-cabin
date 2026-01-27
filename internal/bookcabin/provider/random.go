package provider

import (
	"crypto/rand"
	"math"
	"math/big"
)

type SafeRand struct{}

func NewSafeRand() *SafeRand {
	return &SafeRand{}
}

func (s *SafeRand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(int64(n))
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return int(value.Int64())
}

func (s *SafeRand) Float64() float64 {
	max := new(big.Int).Lsh(big.NewInt(1), 53)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return float64(value.Int64()) / math.Pow(2, 53)
}
