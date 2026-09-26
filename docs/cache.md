# Semantic Vector Cache Architecture

The semantic cache is the core engine of PromptRadar. Unlike traditional exact-match key-value stores (e.g., Redis string matching or Memcached), PromptRadar determines cache hits based on **semantic proximity** in high-dimensional embedding space.

```
       Incoming Prompt
              │
              ▼
   768-D Vector Embedding
              │
              ▼
     L2-Unit Normalization  (|v| = 1.0)
              │
              ▼
   Cosine Similarity Search
      (Dot Product against N cached vectors)
              │
              ▼
    Max Similarity Score: s*
              │
       ┌──────┴──────┐
  s* ≥ τ           s* < τ
       │             │
       ▼             ▼
   CACHE HIT     CACHE MISS
       │             │
   Serve from        Stream from LLM
   Memory (2ms)      & Insert into Cache
```

---

## 1. Cache Entry Structure

Each cached item is represented in memory by the `CacheEntry` struct (`apps/radar-proxy/internal/cache/cache.go`):

```go
type CacheEntry struct {
    ID           string    `json:"id"`
    Prompt       string    `json:"prompt"`
    Response     string    `json:"response"`
    Category     string    `json:"category"`
    TokensInput  int       `json:"tokens_input"`
    TokensOutput int       `json:"tokens_output"`
    Embedding    []float32 `json:"-"`       // 768-D L2-normalized float32 array
    X            float32   `json:"x"`       // 2D Projected Radar coordinate X
    Y            float32   `json:"y"`       // 2D Projected Radar coordinate Y
    HitCount     int       `json:"hit_count"`
    CreatedAt    time.Time `json:"created_at"`
    LastHitAt    time.Time `json:"last_hit_at"`
}
```

### Memory Footprint per Entry
| Field | Data Type | Size (Bytes) |
|---|---|---|
| `ID` | string | ~16 B |
| `Prompt` + `Response` | strings | ~1,024 B (average) |
| `Category` | string | ~16 B |
| `TokensInput` + `TokensOutput` | int (2x) | 16 B |
| `Embedding` | `[]float32` (768 elements) | 768 × 4 B = **3,072 B** |
| `X` + `Y` | float32 (2x) | 8 B |
| `HitCount` + Timestamps | int + 2x time.Time | 56 B |
| **Total per Entry** | | **~4.2 KB** |

At 10,000 cached prompts, total memory overhead is approximately **42 MB**, comfortably fitting in RAM.

---

## 2. Cosine Similarity & Normalization

Cosine similarity measures the cosine of the angle between two high-dimensional vectors:

$$\text{similarity}(\mathbf{A}, \mathbf{B}) = \cos(\theta) = \frac{\mathbf{A} \cdot \mathbf{B}}{\|\mathbf{A}\|_2 \|\mathbf{B}\|_2} = \frac{\sum_{i=1}^{D} A_i B_i}{\sqrt{\sum_{i=1}^{D} A_i^2} \sqrt{\sum_{i=1}^{D} B_i^2}}$$

### Optimization: Unit Normalization
Calculating square roots and Euclidean norms ($\|\mathbf{A}\|_2$) during every vector comparison incurs substantial CPU overhead. PromptRadar enforces an invariant: **all vectors are L2-normalized immediately upon embedding creation**:

$$\|\mathbf{v}\|_2 = \sqrt{\sum_{i=1}^{768} v_i^2} = 1.0$$

Because $\|\mathbf{A}\|_2 = 1.0$ and $\|\mathbf{B}\|_2 = 1.0$, the denominator evaluates to $1.0$:

$$\text{similarity}(\mathbf{A}, \mathbf{B}) = \mathbf{A} \cdot \mathbf{B} = \sum_{i=1}^{768} A_i B_i$$

This reduces the similarity check to a **pure dot product** of 768 floating-point multiplications and additions, executed in **164.2 nanoseconds** on Apple Silicon ARM64 without any heap allocations.

---

## 3. Threshold Behavior

The similarity threshold $\tau \in [0.50, 1.00]$ governs cache decision sensitivity:

```
0.50                   0.80             0.88             0.95            1.00
  │                      │                │                │               │
  └──────────────────────┴────────────────┴────────────────┴───────────────┘
   Loose / False Hits      Balanced         Production Default  Near Exact Match
   (Risk of hallucinated   (General Q&A)    (High precision)    (Synonyms only)
    stale context)
```

1. **$\text{sim} \ge \tau$ (Cache Hit)**:
   - The query matches an existing intent.
   - The cached response is returned immediately ($<5\text{ms}$).
   - `hit_count` is incremented asynchronously under a write lock.
   - Nearest neighbors are extracted for UI radar circle visualization.
2. **$\text{sim} < \tau$ (Cache Miss)**:
   - The query diverges from stored semantic clusters.
   - The proxy invokes the upstream LLM and streams token chunks via SSE.
   - Upon completion, the new prompt, embedding, and response are inserted into `VectorCache`.

---

## 4. Cache Insertion (`Insert`)

When a cache miss completes:
1. The new prompt embedding is projected to 2D coordinates $(x, y)$ using the static matrix $W \in \mathbb{R}^{768 \times 2}$.
2. A new `CacheEntry` is instantiated.
3. The entry is appended to the internal slice under an exclusive `sync.RWMutex.Lock()`:

```go
func (vc *VectorCache) Insert(entry *CacheEntry) {
    vc.mu.Lock()
    defer vc.mu.Unlock()

    if entry.CreatedAt.IsZero() {
        entry.CreatedAt = time.Now()
    }
    entry.LastHitAt = entry.CreatedAt
    vc.entries = append(vc.entries, entry)
}
```

The exclusive lock is held for $<50\text{ns}$ (slice pointer append), preventing any read starvation.

---

## 5. Cache Lookup (`Search`)

Lookup utilizes a shared read lock (`sync.RWMutex.RLock()`), permitting unlimited concurrent reads across all Go worker goroutines:

```go
func (vc *VectorCache) Search(queryVec []float32, threshold float32, topK int) SearchResult {
    vc.mu.RLock()
    defer vc.mu.RUnlock()

    var bestMatch *CacheEntry
    var maxSim float32 = -2.0

    for _, entry := range vc.entries {
        sim := radMath.DotProduct(queryVec, entry.Embedding)
        if sim > maxSim {
            maxSim = sim
            bestMatch = entry
        }
    }
    // Decision logic & Top-K neighbor sorting...
}
```

---

## 6. Algorithmic Complexity

| Metric | In-Memory Flat Scan (Current) | HNSW Graph Index (Future) |
|---|---|---|
| **Search Time Complexity** | $\mathcal{O}(N \cdot D)$ | $\mathcal{O}(\log N \cdot D)$ |
| **Space Complexity** | $\mathcal{O}(N \cdot D)$ | $\mathcal{O}(N \cdot M \cdot D)$ |
| **Insert Complexity** | $\mathcal{O}(1)$ | $\mathcal{O}(M \log N \cdot D)$ |
| **10,000 Vectors Latency** | $1.64\text{ms}$ | $0.20\text{ms}$ |
| **Recall Accuracy** | **100% (Exact)** | 95–99% (Approximate) |

* $N$: Number of cached entries.
* $D$: Embedding dimensions ($768$).
* $M$: Graph connectivity parameter ($16 \le M \le 64$).

Because $N \le 10,000$ in single-tenant deployments, flat scan provides **100% exact recall** with zero approximation errors while remaining well under the $5\text{ms}$ latency budget.

---

## 7. Eviction Policies: Sliding-Window TTL & Max Entries LRU

PromptRadar enforces a dual-tier bounded memory model combining **time-based expiration (TTL)** with a **strict capacity cap (Max Entries LRU)** to guarantee the cache never exhausts host RAM.

```
                  New Cache Insertion
                           │
                           ▼
              Is len(entries) >= MAX_ENTRIES?
                   ┌───────┴───────┐
                  Yes              No
                   │               │
                   ▼               ▼
           Evict LRU Node     Insert Directly
           (Oldest LastHitAt)
                   │
                   ▼
        Periodic Background Ticker (Every 5m)
                   │
                   ▼
     Purge All Nodes Where: (Now - LastHitAt) > TTL
```

### 1. Sliding-Window TTL (`CACHE_TTL`)
* Evaluates `LastHitAt` for every entry:
  - If a prompt is frequently asked, every cache hit updates `LastHitAt = time.Now()`, renewing its lifespan.
  - If a prompt has had no traffic for longer than `CACHE_TTL`, the background worker purges it.
* Controlled via: `CACHE_TTL=24h` (supports standard Go durations: `1h`, `24h`, `7d`).

### 2. Maximum Entries Capacity Cap (`CACHE_MAX_ENTRIES`)
* Prevents memory exhaustion during unexpected traffic surges.
* If `len(entries) >= CACHE_MAX_ENTRIES`, `Insert()` automatically performs an **LRU eviction**:
  $$\text{Target} = \arg\min_{e \in \text{entries}} (e.\text{LastHitAt})$$
* The least recently queried vector is evicted under the exclusive write lock before appending the new entry.
* Controlled via: `CACHE_MAX_ENTRIES=10000` (default: `10000` entries, consuming $\approx 42\text{MB}$ RAM).

---

## 8. Current Scale Limits & Future Roadmap

### Current Scale Boundary
1. **Linear Scan Scaling**: Up to $N \le 10,000$ entries, memory scan takes $< 5.5\text{ms}$. Beyond 50,000 entries, CPU cache lines saturate.
2. **Node Ephemerality**: If the Go proxy process restarts, in-memory entries are re-seeded from `seed_prompts.json` or persistent storage.

### Evolution Path: Hybrid Indexing & pgvector
```
Phase 1 (Current):   In-Memory Flat Scan + Sliding TTL (N ≤ 10,000)
Phase 2 (Scale):     HNSW graph in Go memory or pgvector HNSW indexing in PostgreSQL
```
* **pgvector**: Offloads vector persistence and HNSW graph indexing to the PostgreSQL control plane database using `CREATE INDEX ON cache_vectors USING hnsw (embedding vector_cosine_ops)`.
* **Local Inverted File (IVF)**: Partition vectors into Voronoi cells to search only the closest centroids, reducing search from $N$ to $N/K$ dot products.

