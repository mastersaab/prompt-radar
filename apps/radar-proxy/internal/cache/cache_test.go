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

func TestVectorCache_TTLPruning(t *testing.T) {
	vc := NewVectorCache()

	// 1. Stale entry (last hit 2 hours ago)
	staleEntry := &CacheEntry{
		ID:        "node-stale",
		Prompt:    "Old prompt",
		Response:  "Old answer",
		LastHitAt: time.Now().Add(-2 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	vc.Insert(staleEntry)

	// 2. Fresh entry (last hit just now)
	freshEntry := &CacheEntry{
		ID:        "node-fresh",
		Prompt:    "Fresh prompt",
		Response:  "Fresh answer",
		LastHitAt: time.Now(),
		CreatedAt: time.Now(),
	}
	vc.Insert(freshEntry)

	if vc.Len() != 2 {
		t.Fatalf("expected 2 entries before purge, got %d", vc.Len())
	}

	// Purge with 1-hour TTL
	evicted := vc.PurgeExpired(1 * time.Hour)
	if evicted != 1 {
		t.Fatalf("expected 1 evicted entry, got %d", evicted)
	}

	if vc.Len() != 1 {
		t.Fatalf("expected 1 entry after purge, got %d", vc.Len())
	}

	remaining := vc.GetAll()
	if remaining[0].ID != "node-fresh" {
		t.Fatalf("expected fresh node to remain, got %s", remaining[0].ID)
	}
}

func TestVectorCache_MaxEntriesLRU(t *testing.T) {
	vc := NewVectorCache()
	vc.SetMaxEntries(2) // cap to 2 entries

	e1 := &CacheEntry{ID: "node-1", Prompt: "P1", LastHitAt: time.Now().Add(-10 * time.Minute)}
	e2 := &CacheEntry{ID: "node-2", Prompt: "P2", LastHitAt: time.Now().Add(-5 * time.Minute)}
	vc.Insert(e1)
	vc.Insert(e2)

	if vc.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", vc.Len())
	}

	// Insert third entry: should trigger LRU eviction of node-1
	e3 := &CacheEntry{ID: "node-3", Prompt: "P3", LastHitAt: time.Now()}
	vc.Insert(e3)

	if vc.Len() != 2 {
		t.Fatalf("expected cache to remain capped at 2, got %d", vc.Len())
	}

	entries := vc.GetAll()
	for _, entry := range entries {
		if entry.ID == "node-1" {
			t.Fatalf("expected oldest entry node-1 to be evicted by LRU capacity policy")
		}
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
