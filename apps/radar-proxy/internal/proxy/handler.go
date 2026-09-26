package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prompt-radar/radar-proxy/internal/cache"
	"github.com/prompt-radar/radar-proxy/internal/embedding"
	"github.com/prompt-radar/radar-proxy/internal/llm"
	radMath "github.com/prompt-radar/radar-proxy/internal/math"
	"github.com/prompt-radar/radar-proxy/internal/telemetry"
)

type Server struct {
	cache           *cache.VectorCache
	embedder        embedding.Embedder
	llmClient       llm.Client
	telemetry       *telemetry.Dispatcher
	projectionMat   [][]float32
	defaultThreshold float32
}

func NewServer(
	vc *cache.VectorCache,
	embedder embedding.Embedder,
	llmClient llm.Client,
	telemetry *telemetry.Dispatcher,
	projMat [][]float32,
) *Server {
	return &Server{
		cache:            vc,
		embedder:         embedder,
		llmClient:        llmClient,
		telemetry:        telemetry,
		projectionMat:    projMat,
		defaultThreshold: 0.88,
	}
}

type QueryRequest struct {
	Prompt    string  `json:"prompt"`
	Threshold float32 `json:"threshold"`
	Stream    bool    `json:"stream"`
}

type QueryResponseHit struct {
	Type         string              `json:"type"` // "hit"
	CacheHit     bool                `json:"cache_hit"`
	Similarity   float32             `json:"similarity"`
	Threshold    float32             `json:"threshold"`
	LatencyMs    int64               `json:"latency_ms"`
	BaselineMs   int64               `json:"baseline_ms"`
	TokensSaved  int                 `json:"tokens_saved"`
	CostSaved    float64             `json:"cost_saved"`
	QueryCoord   Coordinate          `json:"query_coord"`
	NearestCoord *Coordinate         `json:"nearest_coord,omitempty"`
	NearestID    string              `json:"nearest_id,omitempty"`
	NearestPrompt string             `json:"nearest_prompt,omitempty"`
	Response     string              `json:"response"`
	Neighbors    []cache.NeighborHit `json:"neighbors"`
}

type Coordinate struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// EnableCORS sets headers for cross-origin requests from the Angular UI.
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HandleQuery processes semantic cache searches and handles hits or streaming misses.
func (s *Server) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request JSON", http.StatusBadRequest)
		return
	}

	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		http.Error(w, "Prompt cannot be empty", http.StatusBadRequest)
		return
	}

	threshold := req.Threshold
	if threshold <= 0 {
		threshold = s.defaultThreshold
	}

	// 1. Generate 768D Embedding
	queryVec, err := s.embedder.Embed(r.Context(), req.Prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Embedding failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 2. Project 2D coordinates for vector radar visualization
	x, y, err := radMath.Project2D(queryVec, s.projectionMat)
	if err != nil {
		x, y = 0, 0
	}

	// 3. Search In-Memory Vector Cache
	searchRes := s.cache.Search(queryVec, threshold, 6)

	queryID := newUUID()
	tokensIn := len(strings.Fields(req.Prompt))
	baselineMs := int64(1250) // baseline un-cached LLM latency

	// Check if this is a CACHE HIT
	if searchRes.Hit && searchRes.BestMatch != nil {
		latencyMs := time.Since(start).Milliseconds()
		if latencyMs == 0 {
			latencyMs = 2 // sub-5ms realistic measure
		}

		tokensSaved := searchRes.BestMatch.TokensInput + searchRes.BestMatch.TokensOutput
		// Model pricing: $0.15 / 1M input, $0.60 / 1M output (~$0.00000045/token avg)
		costSaved := float64(tokensSaved) * 0.00000045

		// Dispatch async telemetry event
		s.telemetry.Dispatch(telemetry.TelemetryEvent{
			QueryID:      queryID,
			Prompt:       req.Prompt,
			CacheHit:     true,
			Similarity:   searchRes.Similarity,
			Threshold:    threshold,
			LatencyMs:    latencyMs,
			BaselineMs:   baselineMs,
			TokensInput:  tokensIn,
			TokensOutput: searchRes.BestMatch.TokensOutput,
			TokensSaved:  tokensSaved,
			CostSaved:    costSaved,
			Model:        s.llmClient.Name(),
			Timestamp:    time.Now(),
		})

		resp := QueryResponseHit{
			Type:         "hit",
			CacheHit:     true,
			Similarity:   searchRes.Similarity,
			Threshold:    threshold,
			LatencyMs:    latencyMs,
			BaselineMs:   baselineMs,
			TokensSaved:  tokensSaved,
			CostSaved:    costSaved,
			QueryCoord:   Coordinate{X: x, Y: y},
			NearestCoord: &Coordinate{X: searchRes.BestMatch.X, Y: searchRes.BestMatch.Y},
			NearestID:    searchRes.BestMatch.ID,
			NearestPrompt: searchRes.BestMatch.Prompt,
			Response:     searchRes.BestMatch.Response,
			Neighbors:    searchRes.Neighbors,
		}

		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if ok {
				payload, _ := json.Marshal(resp)
				fmt.Fprintf(w, "data: %s\n\n", payload)
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// CACHE MISS: Stream from LLM
	var nearestCoord *Coordinate
	var nearestID string
	var nearestPrompt string
	if searchRes.BestMatch != nil {
		nearestCoord = &Coordinate{X: searchRes.BestMatch.X, Y: searchRes.BestMatch.Y}
		nearestID = searchRes.BestMatch.ID
		nearestPrompt = searchRes.BestMatch.Prompt
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 1. Emit Miss Start Event for radar animation trigger
	missStartMeta := map[string]interface{}{
		"type":          "miss_start",
		"cache_hit":     false,
		"similarity":    searchRes.Similarity,
		"threshold":     threshold,
		"query_coord":   Coordinate{X: x, Y: y},
		"nearest_coord": nearestCoord,
		"nearest_id":    nearestID,
		"nearest_prompt": nearestPrompt,
		"neighbors":     searchRes.Neighbors,
	}
	startPayload, _ := json.Marshal(missStartMeta)
	fmt.Fprintf(w, "data: %s\n\n", startPayload)
	flusher.Flush()

	// 2. Stream Tokens from LLM
	chunkCh := make(chan llm.StreamChunk, 20)
	go func() {
		_ = s.llmClient.StreamCompletion(r.Context(), req.Prompt, chunkCh)
	}()

	var fullResponse strings.Builder
	tokensOut := 0

	for chunk := range chunkCh {
		if chunk.Text != "" {
			fullResponse.WriteString(chunk.Text)
			tokenPayload, _ := json.Marshal(map[string]interface{}{
				"type": "token",
				"text": chunk.Text,
			})
			fmt.Fprintf(w, "data: %s\n\n", tokenPayload)
			flusher.Flush()
		}
		if chunk.Done {
			tokensOut = chunk.TokensOut
		}
	}

	totalLatency := time.Since(start).Milliseconds()

	// 3. Store new entry into VectorCache under write lock
	newEntry := &cache.CacheEntry{
		ID:           "cache-" + queryID[:8],
		Prompt:       req.Prompt,
		Response:     fullResponse.String(),
		Category:     detectCategory(req.Prompt),
		TokensInput:  tokensIn,
		TokensOutput: tokensOut,
		Embedding:    queryVec,
		X:            x,
		Y:            y,
		HitCount:     0,
		CreatedAt:    time.Now(),
	}
	s.cache.Insert(newEntry)

	// 4. Emit Miss Complete Event
	missEndMeta := map[string]interface{}{
		"type":         "miss_complete",
		"new_entry_id": newEntry.ID,
		"latency_ms":   totalLatency,
		"tokens_in":    tokensIn,
		"tokens_out":   tokensOut,
	}
	endPayload, _ := json.Marshal(missEndMeta)
	fmt.Fprintf(w, "data: %s\n\n", endPayload)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	// 5. Dispatch async telemetry
	s.telemetry.Dispatch(telemetry.TelemetryEvent{
		QueryID:      queryID,
		Prompt:       req.Prompt,
		CacheHit:     false,
		Similarity:   searchRes.Similarity,
		Threshold:    threshold,
		LatencyMs:    totalLatency,
		BaselineMs:   baselineMs,
		TokensInput:  tokensIn,
		TokensOutput: tokensOut,
		TokensSaved:  0,
		CostSaved:    0,
		Model:        s.llmClient.Name(),
		Timestamp:    time.Now(),
	})
}

// HandleGetNodes returns all cached nodes for radar visualization initialization.
func (s *Server) HandleGetNodes(w http.ResponseWriter, r *http.Request) {
	entries := s.cache.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": len(entries),
		"nodes": entries,
	})
}

// HandleHealth returns proxy operational health and cache statistics.
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "UP",
		"cached_vectors":  s.cache.Len(),
		"embedder":        s.embedder.Name(),
		"llm_engine":      s.llmClient.Name(),
		"telemetry_stats": s.telemetry.Stats(),
	})
}

// SeedWithData populates the cache with initial prompts.
func (s *Server) SeedWithData(ctx context.Context, items []SeedItem) error {
	for _, item := range items {
		vec, err := s.embedder.Embed(ctx, item.Prompt)
		if err != nil {
			continue
		}
		x, y, _ := radMath.Project2D(vec, s.projectionMat)
		s.cache.Insert(&cache.CacheEntry{
			ID:           item.ID,
			Prompt:       item.Prompt,
			Response:     item.Response,
			Category:     item.Category,
			TokensInput:  item.TokensInput,
			TokensOutput: item.TokensOutput,
			Embedding:    vec,
			X:            x,
			Y:            y,
			HitCount:     1,
			CreatedAt:    time.Now(),
		})
	}
	return nil
}

type SeedItem struct {
	ID           string `json:"id"`
	Prompt       string `json:"prompt"`
	Category     string `json:"category"`
	Response     string `json:"response"`
	TokensInput  int    `json:"tokens_input"`
	TokensOutput int    `json:"tokens_output"`
}

func detectCategory(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "go") || strings.Contains(lower, "golang"):
		return "Golang"
	case strings.Contains(lower, "python") || strings.Contains(lower, "pandas"):
		return "Python"
	case strings.Contains(lower, "vector") || strings.Contains(lower, "learning") || strings.Contains(lower, "descent"):
		return "Machine Learning"
	case strings.Contains(lower, "system") || strings.Contains(lower, "raft") || strings.Contains(lower, "cap"):
		return "System Design"
	case strings.Contains(lower, "bread") || strings.Contains(lower, "sourdough") || strings.Contains(lower, "coffee"):
		return "Culinary"
	case strings.Contains(lower, "quantum") || strings.Contains(lower, "physics"):
		return "Physics"
	default:
		return "General"
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
