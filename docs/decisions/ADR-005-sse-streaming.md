# ADR-005: Use Server-Sent Events (SSE) for LLM Inference Streaming

* **Status**: Accepted
* **Deciders**: Systems & Architecture Team
* **Date**: 2026-09-27

---

## 1. Context & Problem Statement
When a cache miss occurs in PromptRadar, the proxy forwards the prompt to the upstream LLM (e.g. Gemini 1.5 Flash). Large language models generate responses autoregressively token-by-token over hundreds to thousands of milliseconds. 

To provide an optimal user experience, tokens must be rendered progressively to the user as they are synthesized, rather than blocking until the full text generation completes. Furthermore, when streaming is complete, the client needs the final telemetry metadata (similarity score, 2D radar coordinates $x, y$, token count, and latency) to update the radar canvas and metrics dashboard in real time.

We needed a streaming protocol that fulfills:
1. Low transport overhead and simple connection management.
2. Standard HTTP semantics compatible with reverse proxies, API gateways, and firewalls.
3. Unidirectional server-to-client data streaming.
4. Native client reconnection and event typing capabilities.

---

## 2. Options Considered

### Option 1: Server-Sent Events (SSE / `text/event-stream`) (Selected)
* **Pros**:
  - **Unidirectional Match**: LLM responses are strictly server-to-client streams; the client sends a single initial prompt and simply listens for incoming tokens.
  - **Standard HTTP/1.1 and HTTP/2**: Uses standard HTTP verbs, headers, and status codes. No connection upgrade protocol negotiation required.
  - **Built-in Browser Support**: Standard `EventSource` API or standard `fetch` with `ReadableStreamDefaultReader` handles the stream with minimal boilerplate.
  - **Native Go Support**: Trivial and zero-allocation streaming via `http.ResponseWriter` and `http.Flusher`.
  - **Event Demultiplexing**: Allows distinct event types (e.g., `event: token` for stream chunks, `event: done` for terminal metadata) over the same connection.
* **Cons**:
  - Unidirectional only (cannot send client messages back over the same socket without opening another HTTP request).
  - Subject to HTTP/1.1 connection limits (max 6 connections per domain in older browsers), though mitigated completely under HTTP/2 multiplexing.

### Option 2: WebSockets (`ws://` / `wss://`)
* **Pros**:
  - Full-duplex bidirectional communication with low per-message framing overhead (2–10 bytes).
* **Cons**:
  - Requires stateful protocol upgrade (`101 Switching Protocols`), which complicates load balancing, sticky sessions, and corporate firewall traversal.
  - LLM token generation does not require bidirectional communication during response generation.
  - Higher connection management complexity (heartbeats, ping/pong frames, stateful reconnect logic).

### Option 3: Polling / Long Polling
* **Pros**:
  - Universal firewall and browser compatibility.
* **Cons**:
  - High latency, severe HTTP connection overhead, and massive server thread exhaustion for token-by-token delivery.

---

## 3. Decision
Use **Server-Sent Events (SSE)** with `Content-Type: text/event-stream` for all LLM inference streaming between `radar-proxy` and `radar-ui`.

The stream lifecycle follows this structured contract:
1. `event: metadata` (immediate): Delivers initial lookup stats (cache hit/miss, similarity score, projected 2D coordinates).
2. `event: token` (repeated): Streams delta text tokens as emitted by the LLM or cache playback.
3. `event: done` (terminal): Delivers total inference latency, token counts, and cost metrics before closing the stream.

---

## 4. Consequences & Trade-offs
* **Positive**:
  - Simple, robust Go implementation using `w.(http.Flusher).Flush()`.
  - Zero connection upgrade failure modes across container networks and enterprise proxies.
  - The client UI uses standard Web Streams (`fetch` + `response.body.getReader()`), seamlessly feeding tokens into the Angular UI and updating the Radar canvas state.
* **Negative**:
  - Client cannot send mid-generation cancellations over the same stream without aborting the `AbortController` (which drops the HTTP connection).

---

## 5. References & Future Alternatives
* [HTML5 W3C Server-Sent Events Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
* **Future Alternative**: If two-way interactive voice or multimodal streaming is introduced, evaluate gRPC-Web or WebSockets with protobuf framing.
