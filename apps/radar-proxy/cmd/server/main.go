package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prompt-radar/radar-proxy/internal/cache"
	"github.com/prompt-radar/radar-proxy/internal/embedding"
	"github.com/prompt-radar/radar-proxy/internal/llm"
	radMath "github.com/prompt-radar/radar-proxy/internal/math"
	"github.com/prompt-radar/radar-proxy/internal/proxy"
	"github.com/prompt-radar/radar-proxy/internal/telemetry"
)

type ProjectionMatrixFile struct {
	Dims          int         `json:"dims"`
	OutDimensions int         `json:"out_dimensions"`
	Matrix        [][]float32 `json:"matrix"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	controlPlaneURL := os.Getenv("CONTROL_PLANE_URL")
	if controlPlaneURL == "" {
		controlPlaneURL = "http://localhost:8080/api/v1/telemetry"
	}

	log.Printf("Starting PromptRadar Data Plane Proxy on :%s...", port)

	// 1. Load Projection Matrix W
	projMatrix := loadProjectionMatrix()
	log.Printf("Loaded projection matrix: %d rows x %d cols", len(projMatrix), len(projMatrix[0]))

	// 2. Initialize Components
	vc := cache.NewVectorCache()
	embedder := embedding.NewEmbedder()
	llmClient := llm.NewClient()
	telemetryDispatcher := telemetry.NewDispatcher(controlPlaneURL)
	defer telemetryDispatcher.Stop()

	server := proxy.NewServer(vc, embedder, llmClient, telemetryDispatcher, projMatrix)

	// 3. Seed Initial Prompts
	seedCache(server)

	// 4. Setup Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/query", server.HandleQuery)
	mux.HandleFunc("/v1/nodes", server.HandleGetNodes)
	mux.HandleFunc("/v1/health", server.HandleHealth)

	handler := proxy.EnableCORS(mux)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// 5. Run Server
	go func() {
		log.Printf("PromptRadar Proxy listening at http://localhost:%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down PromptRadar Data Plane Proxy...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("PromptRadar Data Plane Proxy stopped gracefully.")
}

func loadProjectionMatrix() [][]float32 {
	searchPaths := []string{
		"../../scripts/projection_matrix.json",
		"scripts/projection_matrix.json",
		"../scripts/projection_matrix.json",
		"/app/scripts/projection_matrix.json",
	}

	var data []byte
	var err error
	for _, p := range searchPaths {
		abs, _ := filepath.Abs(p)
		data, err = os.ReadFile(abs)
		if err == nil {
			log.Printf("Found projection matrix at: %s", abs)
			break
		}
	}

	if err != nil {
		log.Println("Projection matrix file not found, generating deterministic fallback matrix...")
		return generateFallbackMatrix()
	}

	var pm ProjectionMatrixFile
	if err := json.Unmarshal(data, &pm); err != nil || len(pm.Matrix) != radMath.Dims {
		log.Printf("Error parsing projection matrix JSON (%v), using fallback", err)
		return generateFallbackMatrix()
	}

	return pm.Matrix
}

func generateFallbackMatrix() [][]float32 {
	mat := make([][]float32, radMath.Dims)
	for i := 0; i < radMath.Dims; i++ {
		// Orthogonal pseudo-random hash projection
		mat[i] = []float32{
			float32(0.08 * (float64((i*17+3)%100)/50.0 - 1.0)),
			float32(0.08 * (float64((i*31+7)%100)/50.0 - 1.0)),
		}
	}
	return mat
}

func seedCache(server *proxy.Server) {
	searchPaths := []string{
		"../../scripts/seed_prompts.json",
		"scripts/seed_prompts.json",
		"../scripts/seed_prompts.json",
		"/app/scripts/seed_prompts.json",
	}

	var data []byte
	var err error
	for _, p := range searchPaths {
		abs, _ := filepath.Abs(p)
		data, err = os.ReadFile(abs)
		if err == nil {
			log.Printf("Found seed prompts at: %s", abs)
			break
		}
	}

	if err != nil {
		log.Printf("Warning: seed_prompts.json not found: %v", err)
		return
	}

	var items []proxy.SeedItem
	if err := json.Unmarshal(data, &items); err != nil {
		log.Printf("Error parsing seed_prompts.json: %v", err)
		return
	}

	ctx := context.Background()
	if err := server.SeedWithData(ctx, items); err != nil {
		log.Printf("Error seeding vector cache: %v", err)
	} else {
		log.Printf("Successfully seeded %d prompts into VectorCache", len(items))
	}
}
