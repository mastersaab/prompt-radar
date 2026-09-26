# PromptRadar API Specification

PromptRadar exposes two distinct API surfaces:
1. **Data Plane API** (Go Proxy, Port `8081`): Low-latency inference, semantic cache queries, SSE token streaming, and radar node inspection.
2. **Control Plane API** (Spring Boot, Port `8080`): Asynchronous batch telemetry ingestion and aggregated analytics.

---

## 1. Data Plane API (Go Proxy — `:8081`)

### 1.1 `POST /v1/query`
Executes semantic vector search against the cache. If similarity $\ge$ threshold, returns a cache hit immediately. If a cache miss occurs, transparently invokes the upstream LLM and streams token responses.

#### Request Headers
| Header | Value | Description |
|---|---|---|
| `Content-Type` | `application/json` | Required |
| `Accept` | `text/event-stream` or `application/json` | Optional. If `text/event-stream` or `"stream": true`, returns SSE. |

#### Request Body
```json
{
  "prompt": "How do I sort a slice of structs by field in Go?",
  "threshold": 0.88,
  "stream": true
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `prompt` | `string` | **Yes** | — | User prompt text to evaluate. Cannot be empty. |
| `threshold` | `float64` | No | `0.85` | Cosine similarity threshold $\tau \in [0.5, 1.0]$. |
| `stream` | `boolean` | No | `true` | If `true`, returns Server-Sent Events stream. |

---

### Response Scenario A: Cache Hit (`stream: false` or JSON Accept)
**HTTP Status:** `200 OK`  
**Content-Type:** `application/json`

```json
{
  "type": "hit",
  "cacheHit": true,
  "similarity": 0.9421,
  "threshold": 0.88,
  "latencyMs": 3,
  "baselineMs": 1250,
  "tokensSaved": 412,
  "costSaved": 0.0001854,
  "queryCoord": {
    "x": 0.312,
    "y": -0.118
  },
  "nearestCoord": {
    "x": 0.305,
    "y": -0.124
  },
  "nearestId": "cache-node-042",
  "nearestPrompt": "Sort slice of structs in Go",
  "response": "In Go 1.21+, use the slices package:\n\n```go\nslices.SortFunc(items, func(a, b Item) int {\n    return cmp.Compare(a.Priority, b.Priority)\n})\n```",
  "neighbors": [
    {
      "id": "cache-node-042",
      "prompt": "Sort slice of structs in Go",
      "similarity": 0.9421,
      "x": 0.305,
      "y": -0.124
    },
    {
      "id": "cache-node-019",
      "prompt": "Custom sort order in Go",
      "similarity": 0.8112,
      "x": 0.280,
      "y": -0.095
    }
  ]
}
```

---

### Response Scenario B: Cache Miss (SSE Streaming)
**HTTP Status:** `200 OK`  
**Content-Type:** `text/event-stream`  
**Cache-Control:** `no-cache`  
**Connection:** `keep-alive`

When a cache miss occurs, the proxy keeps the connection open and streams typed SSE events.

#### Event Stream Protocol

```http
data: {"type":"miss_start","cache_hit":false,"similarity":0.724,"threshold":0.88,"query_coord":{"x":0.512,"y":0.441},"nearest_coord":{"x":0.420,"y":0.310},"nearest_id":"cache-node-011","nearest_prompt":"How to declare variables in Go?","neighbors":[{"id":"cache-node-011","similarity":0.724,"x":0.42,"y":0.31}]}

data: {"type":"token","text":"To sort "}

data: {"type":"token","text":"a slice of structs "}

data: {"type":"token","text":"by field in Go..."}

data: {"type":"miss_complete","new_entry_id":"cache-a7f31c0e","latency_ms":1184,"tokens_in":14,"tokens_out":185}

data: [DONE]
```

#### Event Descriptions
1. **`miss_start`**: Emitted immediately before LLM generation begins. Contains the 768D similarity score, 2D radar coordinates $(x,y)$, and the nearest candidate node for the radar animation to draw the query dot and miss trajectory.
2. **`token`**: Emitted per LLM text chunk as received from Gemini/OpenAI upstream.
3. **`miss_complete`**: Emitted when LLM synthesis is complete and the newly generated response has been committed into the in-memory `VectorCache`.
4. **`[DONE]`**: Standard termination signal indicating the stream is closed.

---

### 1.2 `GET /v1/nodes`
Returns all currently cached prompts, their projected 2D coordinates, hit counters, and metadata for client-side radar canvas rendering.

**HTTP Status:** `200 OK`  
**Content-Type:** `application/json`

#### Response Body
```json
[
  {
    "id": "seed-001",
    "prompt": "How do I implement binary search in Go?",
    "response": "Here is an idiomatic binary search implementation...",
    "category": "Algorithms",
    "tokensInput": 8,
    "tokensOutput": 164,
    "x": 0.452,
    "y": -0.612,
    "hitCount": 14,
    "createdAt": "2026-09-27T00:15:00Z"
  },
  {
    "id": "seed-002",
    "prompt": "Explain database indexing and B-Trees",
    "response": "A B-Tree index maintains sorted key data...",
    "category": "Databases",
    "tokensInput": 6,
    "tokensOutput": 220,
    "x": -0.710,
    "y": 0.334,
    "hitCount": 9,
    "createdAt": "2026-09-27T00:16:30Z"
  }
]
```

---

### 1.3 `GET /v1/health`
Proxy liveness probe.

**HTTP Status:** `200 OK`  
**Content-Type:** `application/json`

```json
{
  "status": "UP",
  "service": "radar-proxy",
  "cacheSize": 128,
  "uptime": "4h12m30s"
}
```

---

## 2. Control Plane API (Spring Boot — `:8080`)

### 2.1 `POST /api/v1/telemetry`
Batch ingestion endpoint called asynchronously by the Go proxy telemetry worker.

**HTTP Status:** `202 Accepted`  
**Content-Type:** `application/json`

#### Request Body
```json
[
  {
    "queryId": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "prompt": "How do I sort a slice in Go?",
    "cacheHit": true,
    "similarity": 0.9421,
    "threshold": 0.88,
    "latencyMs": 3,
    "baselineMs": 1250,
    "tokensInput": 8,
    "tokensOutput": 142,
    "tokensSaved": 150,
    "costSaved": 0.0000675,
    "model": "gemini-1.5-flash",
    "timestamp": "2026-09-27T00:45:00Z"
  }
]
```

#### Response Body
```json
{
  "status": "INGESTED",
  "count": 1
}
```

---

### 2.2 `GET /api/v1/analytics/summary`
Calculates real-time aggregated metrics across all recorded telemetry events.

**HTTP Status:** `200 OK`  
**Content-Type:** `application/json`

#### Response Body
```json
{
  "totalQueries": 1240,
  "cacheHits": 782,
  "cacheMisses": 458,
  "hitRate": 0.6306,
  "totalTokensSaved": 328450,
  "totalLatencySavedMs": 954200,
  "totalCostSaved": 0.1478,
  "avgHitLatencyMs": 3.4,
  "avgMissLatencyMs": 1210.8
}
```

---

### 2.3 `GET /api/v1/analytics/recent`
Returns the latest telemetry logs ordered by timestamp descending.

**Query Parameters:**
* `limit` (optional, default: `20`, max: `100`)

**HTTP Status:** `200 OK`  
**Content-Type:** `application/json`

#### Response Body
```json
[
  {
    "id": 1420,
    "queryId": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "prompt": "How do I sort a slice in Go?",
    "cacheHit": true,
    "similarity": 0.9421,
    "threshold": 0.88,
    "latencyMs": 3,
    "baselineMs": 1250,
    "tokensInput": 8,
    "tokensOutput": 142,
    "tokensSaved": 150,
    "costSaved": 0.0000675,
    "model": "gemini-1.5-flash",
    "createdAt": "2026-09-27T00:45:00Z"
  }
]
```

---

### 2.4 `GET /actuator/health`
Spring Boot health and readiness check.

**HTTP Status:** `200 OK`  
**Content-Type:** `application/json`

```json
{
  "status": "UP",
  "components": {
    "db": {
      "status": "UP",
      "details": {
        "database": "PostgreSQL",
        "validationQuery": "isValid()"
      }
    },
    "diskSpace": {
      "status": "UP"
    }
  }
}
```

---

## 3. Error Responses & HTTP Status Codes

Both services adhere strictly to standard HTTP status codes and return JSON error envelopes when appropriate.

| Status Code | Meaning | When Triggered | Response Payload |
|---|---|---|---|
| `400 Bad Request` | Client Error | Empty prompt, invalid JSON body, or out-of-range threshold. | `{"error": "Prompt cannot be empty"}` |
| `405 Method Not Allowed` | Invalid Verb | `GET` sent to `/v1/query` or `POST` sent to `/v1/nodes`. | `{"error": "Method not allowed"}` |
| `500 Internal Server Error` | Server Exception | Embedding provider API failure or downstream panic. | `{"error": "Embedding failed: upstream timeout"}` |
| `502 Bad Gateway` | Upstream Failure | Upstream LLM provider returned invalid chunk or closed socket. | `{"error": "LLM generation aborted by upstream"}` |
| `503 Service Unavailable` | Overload / Disconnect | Control plane database connection pool exhausted. | `{"error": "Database connection timeout"}` |
