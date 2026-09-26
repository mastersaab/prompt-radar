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
	mu          sync.RWMutex
	entries     []*CacheEntry
	maxEntries  int
	stopCleaner chan struct{}
}

// NewVectorCache creates an empty in-memory vector cache with a default capacity of 10,000.
func NewVectorCache() *VectorCache {
	return &VectorCache{
		entries:     make([]*CacheEntry, 0, 1024),
		maxEntries:  10000,
		stopCleaner: make(chan struct{}, 1),
	}
}

// SetMaxEntries updates the maximum capacity and immediately prunes LRU entries if exceeded.
func (vc *VectorCache) SetMaxEntries(n int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.maxEntries = n
	if n > 0 && len(vc.entries) > n {
		for len(vc.entries) > n {
			vc.evictLRULocked()
		}
	}
}

// evictLRULocked evicts the least recently accessed entry. Must be called while holding write lock.
func (vc *VectorCache) evictLRULocked() {
	if len(vc.entries) == 0 {
		return
	}
	oldestIdx := 0
	oldestTime := vc.entries[0].LastHitAt

	for i := 1; i < len(vc.entries); i++ {
		if vc.entries[i].LastHitAt.Before(oldestTime) {
			oldestTime = vc.entries[i].LastHitAt
			oldestIdx = i
		}
	}

	// Remove entry at oldestIdx
	vc.entries = append(vc.entries[:oldestIdx], vc.entries[oldestIdx+1:]...)
}

// Insert appends a new cache entry under an exclusive write lock.
// If maxEntries is exceeded, the least recently used (LRU) entry is pruned first.
func (vc *VectorCache) Insert(entry *CacheEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.LastHitAt = entry.CreatedAt

	// Evict LRU if capacity reached
	if vc.maxEntries > 0 && len(vc.entries) >= vc.maxEntries {
		vc.evictLRULocked()
	}

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

// PurgeExpired evicts entries that have not been hit within the specified TTL window.
// Returns the count of purged entries.
func (vc *VectorCache) PurgeExpired(ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()

	now := time.Now()
	active := make([]*CacheEntry, 0, len(vc.entries))
	evicted := 0

	for _, entry := range vc.entries {
		if now.Sub(entry.LastHitAt) <= ttl {
			active = append(active, entry)
		} else {
			evicted++
		}
	}

	vc.entries = active
	return evicted
}

// StartTTLCleaner starts a background goroutine that periodically purges expired entries.
func (vc *VectorCache) StartTTLCleaner(ttl time.Duration, interval time.Duration) {
	if ttl <= 0 {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				vc.PurgeExpired(ttl)
			case <-vc.stopCleaner:
				ticker.Stop()
				return
			}
		}
	}()
}

// StopTTLCleaner halts the active TTL background worker.
func (vc *VectorCache) StopTTLCleaner() {
	if vc.stopCleaner != nil {
		select {
		case vc.stopCleaner <- struct{}{}:
		default:
		}
	}
}

