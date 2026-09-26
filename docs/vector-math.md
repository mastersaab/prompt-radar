# Mathematical Foundations of Semantic Radar

PromptRadar balances high-dimensional semantic accuracy with real-time 2D Cartesian rendering. This document details the mathematical framework, vector normalizations, SIMD optimizations, and dimensionality reduction techniques powering the system.

---

## 1. Cosine Similarity & Normalization

Given two embedding vectors $\mathbf{A}, \mathbf{B} \in \mathbb{R}^{D}$ where $D = 768$, the cosine of the angle $\theta$ between them is defined as:

$$\text{similarity}(\mathbf{A}, \mathbf{B}) = \cos(\theta) = \frac{\mathbf{A} \cdot \mathbf{B}}{\|\mathbf{A}\|_2 \|\mathbf{B}\|_2} = \frac{\sum_{i=1}^{D} A_i B_i}{\sqrt{\sum_{i=1}^{D} A_i^2} \sqrt{\sum_{i=1}^{D} B_i^2}}$$

### Why L2-Normalization Enables Direct Dot Product

Computing the square root $\sqrt{\sum A_i^2}$ during search scales poorly under high-throughput request rates. 

PromptRadar enforces an invariant at vector ingestion: every vector is transformed to an **L2-unit vector**:

$$\hat{\mathbf{v}} = \frac{\mathbf{v}}{\|\mathbf{v}\|_2} = \frac{\mathbf{v}}{\sqrt{\sum_{i=1}^{768} v_i^2}}$$

Because both the cached vector $\hat{\mathbf{A}}$ and incoming query vector $\hat{\mathbf{B}}$ have unit length:

$$\|\hat{\mathbf{A}}\|_2 = 1.0, \quad \|\hat{\mathbf{B}}\|_2 = 1.0$$

The denominator simplifies to 1:

$$\text{similarity}(\hat{\mathbf{A}}, \hat{\mathbf{B}}) = \frac{\hat{\mathbf{A}} \cdot \hat{\mathbf{B}}}{1.0 \times 1.0} = \hat{\mathbf{A}} \cdot \hat{\mathbf{B}} = \sum_{i=1}^{768} A_i B_i$$

### Loop Unrolling & Compiler Autovectorization
In `apps/radar-proxy/internal/math/vector.go`, the dot product is unrolled 8 times:

```go
func DotProduct(a, b []float32) float32 {
    var sum float32 = 0
    n := len(a)
    i := 0

    // Unroll 8x for SIMD / NEON / AVX2 register utilization
    for ; i <= n-8; i += 8 {
        sum += a[i]*b[i] +
            a[i+1]*b[i+1] +
            a[i+2]*b[i+2] +
            a[i+3]*b[i+3] +
            a[i+4]*b[i+4] +
            a[i+5]*b[i+5] +
            a[i+6]*b[i+6] +
            a[i+7]*b[i+7]
    }

    for ; i < n; i++ {
        sum += a[i] * b[i]
    }
    return sum
}
```

* **Zero Allocations**: Takes slices directly without allocating heap memory (`0 allocs/op`).
* **Execution Time**: **164.2 nanoseconds** on Apple Silicon M4 ARM64.

---

## 2. Dimensionality Reduction: 768D $\to$ 2D Radar

To render high-dimensional prompts on a 2D HTML5 canvas, PromptRadar maps 768D embeddings into a bounded Cartesian coordinate space $(x, y) \in [-1.0, 1.0]^2$.

```
     768-D Vector v ∈ ℝ⁷⁶⁸
              │
              ▼
   Static Projection Matrix W ∈ ℝ⁷⁶⁸ˣ²
   [ W₁,₁   W₁,₂ ]
   [ W₂,₁   W₂,₂ ]
   [  ...    ...  ]
   [ W₇₆₈,₁ W₇₆₈,₂]
              │
              ▼
    Linear Matrix Product
   x = ∑ vᵢ · Wᵢ,₁
   y = ∑ vᵢ · Wᵢ,₂
              │
              ▼
      Coordinate Clamping
   x, y ∈ [-1.0, 1.0]
              │
              ▼
   Canvas Transformation
   X_canvas = CenterX + x · Radius
   Y_canvas = CenterY - y · Radius
```

### Static Projection Matrix ($W$)
The projection matrix $W \in \mathbb{R}^{768 \times 2}$ is initialized using the top two principal components derived from an offline Principal Component Analysis (PCA) across a representative corpus of 1,000 diverse developer prompts:

$$\mathbf{p} = \hat{\mathbf{v}} \mathbf{W}$$

$$x = \sum_{i=1}^{768} \hat{v}_i W_{i, 0}, \quad y = \sum_{i=1}^{768} \hat{v}_i W_{i, 1}$$

### Why Static Matrix Over Dynamic Online PCA
1. **Coordinate Stability**: In dynamic online PCA, adding a single new vector reorients the covariance eigenvectors, causing existing nodes to shift erratically on the screen.
2. **Deterministic Positioning**: A prompt will always project to the exact same $(x, y)$ coordinate across all clients and server restarts.
3. **Zero Latency Overhead**: Linear projection requires only 1,536 floating-point operations ($<0.5\mu\text{s}$).

---

## 3. The Core System Invariant

> [!IMPORTANT]
> **2D coordinates are visualization only; cache decisions happen strictly in 768D.**

Dimensionality reduction from 768 dimensions down to 2 dimensions inevitably incurs significant information loss (Johnson-Lindenstrauss lemma lower bounds). 

### Why 2D Distance Cannot Determine Cache Hits
In high dimensions, orthogonal vectors can collapse onto nearby 2D points under projection:
* Two prompts with completely different intents (e.g. `How to bake bread` and `Kubernetes pod restart policy`) might project near each other if their projections along the top two principal axes overlap.
* If cache decisions were made based on 2D Euclidean distance on the canvas ($\sqrt{\Delta x^2 + \Delta y^2}$), the cache would suffer catastrophic false positive rates.

### Mathematical Separation of Concerns
| Component | Metric Used | Dimension | Purpose |
|---|---|---|---|
| **Cache Decision** | Normalized Dot Product (Cosine) | **768D** | Guarantees semantic precision and prevents hallucinated/irrelevant cache hits. |
| **Radar Rendering** | Cartesian $(x, y)$ Projection | **2D** | Human-interpretable cluster visualization, hit rings, and trajectory animations. |

---

## 4. Converting Similarity to Radar Threshold Radii

On the radar canvas, each cached node is rendered with a circular threshold halo representing the sensitivity threshold $\tau$.

Since cosine similarity measures angular divergence, the angular distance in radians is:

$$\theta = \arccos(\text{similarity})$$

To render the threshold boundary in canvas pixel space:

$$R_{\text{threshold}} = R_{\text{max}} \times (1.0 - \tau) \times K_{\text{scale}}$$

* When $\tau = 1.0$ (exact match only), $R_{\text{threshold}} = 0$ (a single point).
* When $\tau = 0.85$, $R_{\text{threshold}}$ forms a halo of radius $15\%$ of the maximum neighborhood scale.
* An incoming query dot lying visually inside the halo indicates that its angular distance is within the accepted threshold boundary.
