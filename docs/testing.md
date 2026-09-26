# Testing Strategy & Verification Suite

PromptRadar enforces a multi-tier testing strategy covering unit math verification, integration contracts, end-to-end user flows, microbenchmarks, and edge-case failure modes.

```
       ┌───────────────────────────────┐
       │   End-to-End (E2E) Tests      │  Full UI ↔ Proxy ↔ Control Plane
       └───────────────┬───────────────┘
                       │
       ┌───────────────▼───────────────┐
       │      Integration Tests        │  HTTP Handlers, JPA DB, SSE Streams
       └───────────────┬───────────────┘
                       │
       ┌───────────────▼───────────────┐
       │         Unit Tests            │  SIMD Math, Cache Search, Normalization
       └───────────────┬───────────────┘
                       │
       ┌───────────────▼───────────────┐
       │       Microbenchmarks         │  Cosine Similarity, 10K Search Latency
       └───────────────────────────────┘
```

---

## 1. Test Categories & Execution

### 1.1 Unit Tests (Go Data Plane)
Validates core vector operations, normalization invariants, projection clamping, and deterministic semantic hashing.

```bash
cd apps/radar-proxy
go test -v -race ./internal/...
```

* `vector_test.go`: Verifies loop unrolling, L2 unit norm invariant ($\|\mathbf{v}\|_2 \approx 1.0$), and 2D bounds clamping ($[-1.0, 1.0]$).
* `cache_test.go`: Tests exact hit lookup, distant miss lookup, and concurrent slice modification.
* `embedder_test.go`: Validates deterministic fallback embeddings and dimension conformity (768D).

### 1.2 Benchmark Tests (Go)
Ensures zero-allocation targets and sub-millisecond execution budgets are never breached during refactoring:

```bash
cd apps/radar-proxy
go test -bench=. -benchmem ./internal/...
```

### 1.3 Spring Boot Integration Tests
Validates repository queries, Flyway schema migrations, and REST endpoints:

```bash
cd apps/radar-control-plane
./gradlew test
```

### 1.4 Frontend Tests (Angular)
Verifies Canvas lifecycle hooks, signal bindings, and SSE stream parsing:

```bash
cd apps/radar-ui
npm test -- --watch=false --browsers=ChromeHeadless
```

---

## 2. Critical Scenario Test Matrix

| # | Test Scenario | Input / Trigger | Expected Behavior | Verification Assert |
|---|---|---|---|---|
| **1** | **Exact Duplicate** | Identical prompt string submitted twice: `"How do I sort a slice in Go?"` | Second query triggers an instantaneous cache hit ($<5\text{ms}$). | `cache_hit: true`, `similarity: 1.00`, 0 LLM API calls. |
| **2** | **Semantic Duplicate** | Reworded prompt: `"Sort slice of structs in Go"` vs cached `"How to sort struct slice in Go language?"` | Semantic proximity recognized. High cosine score ($\ge 0.90$). | `cache_hit: true`, `similarity ≥ threshold`, returns cached response. |
| **3** | **Similar But Different** | `"How do I sort a slice in Go?"` vs `"How do I sort a map in Go?"` | Vector diverges due to orthogonal intent. | `cache_hit: false`, `similarity < threshold`, triggers LLM. |
| **4** | **Threshold Boundary** | Configured $\tau = 0.880$. Case A: $\text{sim} = 0.881$. Case B: $\text{sim} = 0.879$. | Case A registers a hit; Case B registers a miss. | Strict monotonic inequality check ($s \ge \tau$). |
| **5** | **Empty / Blank Prompt** | Body contains `{"prompt": "   "}`. | Request is rejected before embedding generation. | HTTP `400 Bad Request`, `{"error": "Prompt cannot be empty"}`. |
| **6** | **Embedding API Failure** | Google Embedding endpoint returns `503` or invalid JSON. | Falls back to deterministic semantic embedder; proxy remains operational. | Request succeeds; logs warning without crashing. |
| **7** | **LLM API Failure / 500** | Upstream Gemini API returns rate limit or quota exceeded. | Proxy emits error event or graceful fallback response. | SSE stream sends error event, closes connection cleanly. |
| **8** | **LLM Timeout** | Upstream LLM fails to deliver first token within 15 seconds. | Proxy context cancels; flushes error message to client. | Context deadline exceeded; goroutine cleaned up. |
| **9** | **Client SSE Disconnect** | Browser tab closed or refreshed mid-stream. | Go handler detects broken pipe via `r.Context().Done()` and aborts LLM loop. | Upstream streaming aborted; socket handles freed. |
| **10**| **Spring Boot Down** | Control plane killed or offline during data plane execution. | Proxy enqueues telemetry; non-blocking select sheds queue when full. | Proxy continues serving queries at 3ms; zero crashes. |
| **11**| **Cache Concurrency** | 100 concurrent reading goroutines while 10 writes insert new entries. | `sync.RWMutex` serializes writes without corrupting memory or deadlocking. | `go test -race` passes with zero data race warnings. |

---

## 3. Load & Stress Testing

To simulate production traffic spikes, execute the following `k6` load test:

```javascript
// loadtest.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp up to 50 users
    { duration: '1m', target: 200 },  // Hold at 200 concurrent users
    { duration: '30s', target: 0 },   // Ramp down
  ],
};

const prompts = [
  "How to implement binary search in Go?",
  "Explain B-Tree indexes in database engines",
  "Write an idiomatic HTTP middleware in Go",
  "What is the difference between concurrency and parallelism?"
];

export default function () {
  const prompt = prompts[Math.floor(Math.random() * prompts.length)];
  const res = http.post('http://localhost:8081/v1/query', JSON.stringify({
    prompt: prompt,
    threshold: 0.88,
    stream: false
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'latency < 20ms': (r) => r.timings.duration < 20,
  });
  sleep(0.05);
}
```

```bash
k6 run loadtest.js
```
