# ADR-004: Static Projection Matrix for 2D Visualization and Dimensionality Decoupling

* **Status**: Accepted
* **Deciders**: Systems & Architecture Team
* **Date**: 2026-09-27

---

## 1. Context & Problem Statement
The 2D Vector Radar renders embeddings on a 2D canvas at 60 FPS. High-dimensional embeddings ($768\text{D}$) must be mapped to screen coordinates $(x, y) \in [-1.0, 1.0]$.

Key challenges:
1. **Coordinate Drift**: If PCA is recomputed dynamically as new prompts arrive, principal components rotate, causing all existing nodes on the client canvas to jump and shift positions unpredictably.
2. **Computational Overhead**: Recomputing t-SNE or UMAP per request is $O(N^2)$ and introduces hundreds of milliseconds of latency.
3. **Semantic Truth vs. Visual Representation**: Projecting 768 dimensions down to 2 dimensions inevitably discards variance. Cache decisions must never be evaluated in 2D space.

---

## 2. Options Considered

### Option 1: Static Precomputed Projection Matrix $\mathbf{W} \in \mathbb{R}^{768 \times 2}$ (Selected)
* **Design**:
  - An offline script computes an orthogonal projection matrix $\mathbf{W}$ (using Johnson-Lindenstrauss lemma / initial PCA basis).
  - Screen coordinates are calculated in Go via simple matrix-vector multiplication:
    $$\begin{bmatrix} x \\ y \end{bmatrix} = \mathbf{u}_{1 \times 768} \times \mathbf{W}_{768 \times 2}$$
* **Pros**:
  - Blazing fast: $768 \times 2 = 1,536$ multiplications ($\approx 1\mu\text{s}$).
  - Completely deterministic: A prompt will always map to the exact same $(x, y)$ coordinate across all sessions and server restarts.
  - Zero coordinate drift when new items are added to the cache.
* **Cons**:
  - Static 2D projection does not capture non-linear manifold relationships as dynamically as t-SNE.

### Option 2: Dynamic Per-Query PCA
* **Pros**: Maximizes variance across the current active dataset.
* **Cons**: Every new query changes the covariance matrix and rotates principal eigenvectors, causing all cached nodes on the radar to jitter and relocate.

### Option 3: Evaluating Cache Decisions in 2D Space (Rejected)
* **Why Rejected**: Mapping $768\text{D} \to 2\text{D}$ discards over $90\%$ of semantic variance. Two completely unrelated prompts might happen to collide or map close together in 2D space while being orthogonal in 768D space. Evaluating cache hits in 2D would cause severe semantic false positives.

---

## 3. Decision
1. **Semantic Truth strictly resides in 768-D space**: All cache hit/miss decisions are evaluated using 768-D unit vector cosine similarities.
2. **Visual Coordinates use Static Projection Matrix $\mathbf{W}$**: 2D screen coordinates $(x, y)$ are computed deterministically via $\mathbf{u} \times \mathbf{W}$ strictly for visualization.

---

## 4. Consequences & Trade-offs
* **Positive**:
  - Guaranteed visual stability across client sessions.
  - Zero false positives in cache decisions.
  - Sub-microsecond projection overhead.
* **Negative**:
  - Clustered nodes on the radar canvas may occasionally overlap visually in 2D despite being distinctly separated in 768-D space. Addressed by category color coding and interactive tooltips.
