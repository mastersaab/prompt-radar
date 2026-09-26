package embedding

import (
	"context"
	"testing"

	radMath "github.com/prompt-radar/radar-proxy/internal/math"
)

func TestSemanticEmbedderParaphraseSimilarity(t *testing.T) {
	embedder := NewSemanticEmbedder()
	ctx := context.Background()

	p1 := "How to sort a slice in Go?"
	p2 := "Golang sort slice example"
	p3 := "How to bake a sourdough bread loaf at home"

	v1, err := embedder.Embed(ctx, p1)
	if err != nil {
		t.Fatalf("Embed p1 failed: %v", err)
	}

	v2, err := embedder.Embed(ctx, p2)
	if err != nil {
		t.Fatalf("Embed p2 failed: %v", err)
	}

	v3, err := embedder.Embed(ctx, p3)
	if err != nil {
		t.Fatalf("Embed p3 failed: %v", err)
	}

	simParaphrase := radMath.DotProduct(v1, v2)
	simUnrelated := radMath.DotProduct(v1, v3)

	t.Logf("Similarity (Paraphrase: '%s' vs '%s'): %.4f", p1, p2, simParaphrase)
	t.Logf("Similarity (Unrelated: '%s' vs '%s'): %.4f", p1, p3, simUnrelated)

	if simParaphrase < 0.85 {
		t.Fatalf("expected high paraphrase similarity >= 0.85, got %.4f", simParaphrase)
	}

	if simUnrelated > 0.40 {
		t.Fatalf("expected low unrelated similarity <= 0.40, got %.4f", simUnrelated)
	}
}
