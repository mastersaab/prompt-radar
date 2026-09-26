package cache

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	radMath "github.com/prompt-radar/radar-proxy/internal/math"
)

func generateRandomNormalizedVec(rng *rand.Rand) []float32 {
	v := make([]float32, radMath.Dims)
	for i := 0; i < radMath.Dims; i++ {
		v[i] = rng.Float32()
	}
	return radMath.Normalize(v)
}

func populateTestCache(n int) (*VectorCache, []float32) {
	rng := rand.New(rand.NewSource(1337))
	vc := NewVectorCache()

	var firstVec []float32
	for i := 0; i < n; i++ {
		vec := generateRandomNormalizedVec(rng)
		if i == 0 {
			firstVec = vec
		}
		vc.Insert(&CacheEntry{
			ID:           fmt.Sprintf("node-%d", i),
			Prompt:       fmt.Sprintf("Prompt index %d", i),
			Response:     fmt.Sprintf("Response index %d", i),
			Category:     "Testing",
			TokensInput:  10,
			TokensOutput: 50,
			Embedding:    vec,
			X:            rng.Float32()*2 - 1,
			Y:            rng.Float32()*2 - 1,
			HitCount:     0,
			CreatedAt:    time.Now(),
		})
	}

	return vc, firstVec
}

func TestVectorCache_HitAndMiss(t *testing.T) {
	vc, hitVec := populateTestCache(100)

	// Test Exact Hit
	res := vc.Search(hitVec, 0.99, 5)
	if !res.Hit {
		t.Fatalf("expected cache hit for exact vector, got miss with sim: %f", res.Similarity)
	}
	if res.Similarity < 0.999 {
		t.Fatalf("expected near 1.0 similarity for self, got %f", res.Similarity)
	}

	// Test Definite Miss
	rng := rand.New(rand.NewSource(9999))
	missVec := generateRandomNormalizedVec(rng)
	resMiss := vc.Search(missVec, 0.99, 5)
	if resMiss.Hit {
		t.Fatalf("expected cache miss for random vector, got hit with sim: %f", resMiss.Similarity)
	}
}

func BenchmarkVectorSearch1K(b *testing.B) {
	vc, queryVec := populateTestCache(1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = vc.Search(queryVec, 0.88, 5)
	}
}

func BenchmarkVectorSearch5K(b *testing.B) {
	vc, queryVec := populateTestCache(5000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = vc.Search(queryVec, 0.88, 5)
	}
}

func BenchmarkVectorSearch10K(b *testing.B) {
	vc, queryVec := populateTestCache(10000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = vc.Search(queryVec, 0.88, 5)
	}
}
