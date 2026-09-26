package cache

import (
	"sort"
	"sync"
	"time"

	radMath "github.com/prompt-radar/radar-proxy/internal/math"
)

// CacheEntry represents a cached prompt-response pair with vector coordinates.
type CacheEntry struct {
	ID           string    `json:"id"`
	Prompt       string    `json:"prompt"`
	Response     string    `json:"response"`
	Category     string    `json:"category"`
	TokensInput  int       `json:"tokens_input"`
	TokensOutput int       `json:"tokens_output"`
	Embedding    []float32 `json:"-"`
	X            float32   `json:"x"`
	Y            float32   `json:"y"`
	HitCount     int       `json:"hit_count"`
	CreatedAt    time.Time `json:"created_at"`
	LastHitAt    time.Time `json:"last_hit_at"`
}

// SearchResult bundles the search decision and visualization details.
type SearchResult struct {
	Hit         bool          `json:"hit"`
	BestMatch   *CacheEntry   `json:"best_match,omitempty"`
	Similarity  float32       `json:"similarity"`
	Threshold   float32       `json:"threshold"`
	Neighbors   []NeighborHit `json:"neighbors"`
	TotalCached int           `json:"total_cached"`
}

// NeighborHit provides surrounding nodes with distances for the radar canvas.
type NeighborHit struct {
	ID         string  `json:"id"`
	Prompt     string  `json:"prompt"`
	Category   string  `json:"category"`
	Similarity float32 `json:"similarity"`
	X          float32 `json:"x"`
	Y          float32 `json:"y"`
}

// VectorCache is an in-memory concurrent vector cache protected by sync.RWMutex.
type VectorCache struct {
	mu      sync.RWMutex
	entries []*CacheEntry
}

// NewVectorCache creates an empty in-memory vector cache.
func NewVectorCache() *VectorCache {
	return &VectorCache{
		entries: make([]*CacheEntry, 0, 1024),
	}
}

// Insert appends a new cache entry under an exclusive write lock.
func (vc *VectorCache) Insert(entry *CacheEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.LastHitAt = entry.CreatedAt
	vc.entries = append(vc.entries, entry)
}

// Search scans all vectors in memory using dot products under a shared read lock (RLock).
func (vc *VectorCache) Search(queryVec []float32, threshold float32, topK int) SearchResult {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	res := SearchResult{
		Hit:         false,
		Threshold:   threshold,
		TotalCached: len(vc.entries),
		Neighbors:   make([]NeighborHit, 0, topK),
	}

	if len(vc.entries) == 0 {
		return res
	}

	type scoredEntry struct {
		entry *CacheEntry
		sim   float32
	}

	scores := make([]scoredEntry, len(vc.entries))
	var bestMatch *CacheEntry
	var maxSim float32 = -2.0

	for i, entry := range vc.entries {
		sim := radMath.DotProduct(queryVec, entry.Embedding)
		scores[i] = scoredEntry{entry: entry, sim: sim}
		if sim > maxSim {
			maxSim = sim
			bestMatch = entry
		}
	}

	res.Similarity = maxSim
	if maxSim >= threshold && bestMatch != nil {
		res.Hit = true
		res.BestMatch = bestMatch
		// Atomic-like update under write lock if hit count needs incrementing
		go vc.incrementHitCount(bestMatch.ID)
	} else if bestMatch != nil {
		res.BestMatch = bestMatch // keep nearest for visualization radar circle
	}

	// Sort neighbors descending by similarity
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].sim > scores[j].sim
	})

	limit := topK
	if limit > len(scores) {
		limit = len(scores)
	}

	for i := 0; i < limit; i++ {
		res.Neighbors = append(res.Neighbors, NeighborHit{
			ID:         scores[i].entry.ID,
			Prompt:     scores[i].entry.Prompt,
			Category:   scores[i].entry.Category,
			Similarity: scores[i].sim,
			X:          scores[i].entry.X,
			Y:          scores[i].entry.Y,
		})
	}

	return res
}

func (vc *VectorCache) incrementHitCount(id string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	for _, e := range vc.entries {
		if e.ID == id {
			e.HitCount++
			e.LastHitAt = time.Now()
			break
		}
	}
}

// GetAll returns a snapshot of all cached entries for radar visualizer initialization.
func (vc *VectorCache) GetAll() []*CacheEntry {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	snapshot := make([]*CacheEntry, len(vc.entries))
	copy(snapshot, vc.entries)
	return snapshot
}

// Len returns the current number of cached entries.
func (vc *VectorCache) Len() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return len(vc.entries)
}
