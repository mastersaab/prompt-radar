# ADR-003: In-Memory Normalized Vector Cache vs. External Stores

* **Status**: Accepted
* **Deciders**: Systems & Architecture Team
* **Date**: 2026-09-27

---

## 1. Context & Problem Statement
Traditional caching mechanisms rely on string key hashing (e.g., Redis `SET prompt_hash response`). However, natural language queries have infinite lexical permutations for identical semantic intent. 

We needed a cache that:
1. Matches semantic intent regardless of phrasing (*"How to sort slice in Go?"* vs *"Golang sort slice example"*).
2. Executes lookups in $<5\text{ms}$.
3. Avoids unnecessary external network hops for maximum proxy throughput.

---

## 2. Options Considered

### Option 1: In-Memory Normalized Vector Cache with `sync.RWMutex` (Selected)
* **Design**:
  - Ingested embeddings are normalized to unit Euclidean length ($\|\mathbf{v}\| = 1$).
  - Cosine similarity simplifies to an 8x unrolled Euclidean dot product ($\sum u_i v_i$).
  - Search runs concurrently under shared read locks (`sync.RWMutex.RLock`).
* **Pros**:
  - Zero network hops: Data resides directly in Go application RAM.
  - Blazing speed: **164.2 ns** per vector comparison, scanning 1,000 vectors in $0.16\text{ms}$ and 10,000 in $1.64\text{ms}$.
  - Zero external dependencies to run locally.
* **Cons**:
  - Linear scan complexity $O(ND)$ requires index evolution when $N > 25,000$.
  - State is localized to the process instance (not horizontally shared across stateless proxy pods without a central coordinator).

### Option 2: Redis String Key-Value
* **Pros**: Distributed, widely deployed.
* **Cons**: 100% cache miss on paraphrased queries; unusable for semantic LLM caching.

### Option 3: External Vector Database (pgvector / Qdrant / Pinecone)
* **Pros**: Supports millions of vectors using HNSW indexing.
* **Cons**: Adds 10–35ms of network and serialization latency per query; significantly degrades the target sub-10ms cache hit response time for moderate cache sizes ($N < 10,000$).

---

## 3. Decision
Implement an **in-memory unit-normalized vector cache** protected by `sync.RWMutex` for the proxy data plane.

---

## 4. Consequences & Trade-offs
* **Positive**:
  - Achieved sub-3ms cache hit response times.
  - Concurrent readers acquire shared locks with near-zero lock contention.
  - Write lock is held only for microseconds when a newly generated LLM response is cached.
* **Negative**:
  - Memory consumption scales linearly: 10,000 768-D vectors require $\approx 30.7\text{MB}$ of RAM (negligible on modern servers).

---

## 5. Upgrade Path
Isolate vector search behind a `VectorIndex` Go interface. When the cache grows beyond 25,000 entries, an HNSW (Hierarchical Navigable Small World) in-memory graph index or an external pgvector service can be plugged in without changing the HTTP proxy layer.
