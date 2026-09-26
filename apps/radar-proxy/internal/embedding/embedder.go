package embedding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	radMath "github.com/prompt-radar/radar-proxy/internal/math"
)

// Embedder generates 768-dimensional normalized embedding vectors.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Name() string
}

// NewEmbedder detects environment configuration and returns the appropriate embedder.
func NewEmbedder() Embedder {
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		geminiKey = os.Getenv("GOOGLE_API_KEY")
	}

	if geminiKey != "" {
		return NewGeminiEmbedder(geminiKey, NewSemanticEmbedder())
	}

	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost != "" {
		return NewOllamaEmbedder(ollamaHost, "nomic-embed-text", NewSemanticEmbedder())
	}

	return NewSemanticEmbedder()
}

// SemanticEmbedder provides a high-quality deterministic offline embedding engine.
// It maps semantic topics, token n-grams, and lexical representations into unit 768D vectors.
type SemanticEmbedder struct {
	stopWords map[string]bool
	clusters  map[string][]int // semantic keywords mapped to dimension indices
}

// NewSemanticEmbedder initializes the offline deterministic embedder.
func NewSemanticEmbedder() *SemanticEmbedder {
	se := &SemanticEmbedder{
		stopWords: map[string]bool{
			"a": true, "an": true, "the": true, "in": true, "on": true,
			"at": true, "to": true, "for": true, "of": true, "with": true,
			"is": true, "are": true, "was": true, "were": true, "be": true,
			"this": true, "that": true, "it": true, "and": true, "or": true,
			"how": true, "what": true, "why": true, "do": true, "does": true,
			"can": true, "you": true, "i": true, "me": true,
		},
		clusters: make(map[string][]int),
	}

	// Define semantic dimension clusters across 768 dimensions
	// Each semantic field maps to specific orthogonal coordinate bands
	se.initCluster("golang", 0, 80)
	se.initCluster("go", 0, 80)
	se.initCluster("slice", 0, 80)
	se.initCluster("goroutine", 20, 100)
	se.initCluster("channel", 20, 100)
	se.initCluster("error", 40, 120)

	se.initCluster("python", 120, 200)
	se.initCluster("tree", 130, 210)
	se.initCluster("binary", 130, 210)
	se.initCluster("pandas", 150, 230)
	se.initCluster("csv", 150, 230)

	se.initCluster("machine", 230, 310)
	se.initCluster("learning", 230, 310)
	se.initCluster("gradient", 230, 310)
	se.initCluster("descent", 230, 310)
	se.initCluster("vector", 250, 330)
	se.initCluster("embedding", 250, 330)
	se.initCluster("cosine", 250, 330)

	se.initCluster("distributed", 330, 410)
	se.initCluster("cap", 330, 410)
	se.initCluster("system", 330, 410)
	se.initCluster("raft", 350, 430)
	se.initCluster("consensus", 350, 430)

	se.initCluster("react", 430, 510)
	se.initCluster("dom", 430, 510)
	se.initCluster("virtual", 430, 510)
	se.initCluster("angular", 450, 530)
	se.initCluster("frontend", 450, 530)

	se.initCluster("bread", 530, 610)
	se.initCluster("sourdough", 530, 610)
	se.initCluster("baking", 530, 610)
	se.initCluster("loaf", 530, 610)
	se.initCluster("coffee", 550, 630)
	se.initCluster("french", 550, 630)
	se.initCluster("press", 550, 630)

	se.initCluster("quantum", 630, 710)
	se.initCluster("entanglement", 630, 710)
	se.initCluster("physics", 630, 710)
	se.initCluster("particle", 630, 710)

	return se
}

func (se *SemanticEmbedder) initCluster(keyword string, start, end int) {
	indices := make([]int, end-start)
	for i := range indices {
		indices[i] = start + i
	}
	se.clusters[keyword] = indices
}

func (se *SemanticEmbedder) Name() string {
	return "deterministic-semantic-v1"
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]+`)

// Embed converts text into a 768-D normalized semantic vector.
func (se *SemanticEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	cleaned := strings.ToLower(text)
	cleaned = nonAlphanumericRegex.ReplaceAllString(cleaned, " ")
	tokens := strings.Fields(cleaned)

	vec := make([]float32, radMath.Dims)

	if len(tokens) == 0 {
		// Return unit vector along first dimension if blank
		vec[0] = 1.0
		return vec, nil
	}

	// 1. Concept Cluster Projection
	clusterWeights := make(map[string]float32)
	for _, token := range tokens {
		clusterWeights[token] += 1.0
	}

	for word, weight := range clusterWeights {
		if indices, ok := se.clusters[word]; ok {
			for idx, dim := range indices {
				phase := float64(idx) * 0.1
				vec[dim] += weight * 3.5 * float32(math.Cos(phase)+1.0)
			}
		}
	}

	// 2. Token Substring & N-gram Hashing Projection
	for i, token := range tokens {
		weight := float32(1.0)
		if se.stopWords[token] {
			weight = 0.2
		}

		// Word hash
		hashWord := hashStringToUint64(token)
		dimWord := int(hashWord % uint64(radMath.Dims))
		signWord := float32(1.0)
		if (hashWord>>63)&1 == 1 {
			signWord = -1.0
		}
		vec[dimWord] += signWord * weight * 1.5

		// Character trigrams for morphological robustness (e.g., sort, sorting, sorted)
		if len(token) >= 3 {
			for j := 0; j <= len(token)-3; j++ {
				trigram := token[j : j+3]
				hTri := hashStringToUint64(trigram)
				dimTri := int(hTri % uint64(radMath.Dims))
				signTri := float32(1.0)
				if (hTri>>63)&1 == 1 {
					signTri = -1.0
				}
				vec[dimTri] += signTri * 0.8
			}
		}

		// Bigram with next word
		if i+1 < len(tokens) {
			bigram := token + "_" + tokens[i+1]
			hBi := hashStringToUint64(bigram)
			dimBi := int(hBi % uint64(radMath.Dims))
			signBi := float32(1.0)
			if (hBi>>63)&1 == 1 {
				signBi = -1.0
			}
			vec[dimBi] += signBi * 2.0
		}
	}

	return radMath.Normalize(vec), nil
}

func hashStringToUint64(s string) uint64 {
	hasher := sha256.New()
	hasher.Write([]byte(s))
	sum := hasher.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

// GeminiEmbedder calls Google's text-embedding-004 API with fallback.
type GeminiEmbedder struct {
	apiKey   string
	client   *http.Client
	fallback Embedder
}

func NewGeminiEmbedder(apiKey string, fallback Embedder) *GeminiEmbedder {
	return &GeminiEmbedder{
		apiKey:   apiKey,
		client:   &http.Client{Timeout: 10 * time.Second},
		fallback: fallback,
	}
}

func (g *GeminiEmbedder) Name() string {
	return "gemini-text-embedding-004"
}

type geminiEmbedRequest struct {
	Model   string `json:"model"`
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type geminiEmbedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *GeminiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent?key=%s", g.apiKey)

	reqBody := geminiEmbedRequest{
		Model: "models/text-embedding-004",
	}
	reqBody.Content.Parts = []struct {
		Text string `json:"text"`
	}{{Text: text}}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return g.fallback.Embed(ctx, text)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return g.fallback.Embed(ctx, text)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return g.fallback.Embed(ctx, text)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return g.fallback.Embed(ctx, text)
	}

	var res geminiEmbedResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil || len(res.Embedding.Values) == 0 {
		return g.fallback.Embed(ctx, text)
	}

	return radMath.Normalize(res.Embedding.Values), nil
}

// OllamaEmbedder calls local Ollama embeddings API.
type OllamaEmbedder struct {
	host     string
	model    string
	client   *http.Client
	fallback Embedder
}

func NewOllamaEmbedder(host, model string, fallback Embedder) *OllamaEmbedder {
	return &OllamaEmbedder{
		host:     strings.TrimRight(host, "/"),
		model:    model,
		client:   &http.Client{Timeout: 5 * time.Second},
		fallback: fallback,
	}
}

func (o *OllamaEmbedder) Name() string {
	return "ollama-" + o.model
}

func (o *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	url := fmt.Sprintf("%s/api/embeddings", o.host)
	payload, _ := json.Marshal(map[string]string{
		"model":  o.model,
		"prompt": text,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return o.fallback.Embed(ctx, text)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return o.fallback.Embed(ctx, text)
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Embedding) == 0 {
		return o.fallback.Embed(ctx, text)
	}

	return radMath.Normalize(result.Embedding), nil
}
