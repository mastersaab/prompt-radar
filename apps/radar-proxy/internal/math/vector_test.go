package math

import (
	"math/rand"
	"testing"
)

func TestDotProductAndNormalization(t *testing.T) {
	v1 := make([]float32, Dims)
	v2 := make([]float32, Dims)

	for i := 0; i < Dims; i++ {
		v1[i] = 1.0
		v2[i] = 1.0
	}

	norm1 := Normalize(v1)
	norm2 := Normalize(v2)

	// Since both are identical unit vectors, DotProduct must equal 1.0
	sim := DotProduct(norm1, norm2)
	if sim < 0.999 || sim > 1.001 {
		t.Fatalf("expected similarity ~1.0 for identical unit vectors, got %f", sim)
	}

	// Orthogonal vector test
	v3 := make([]float32, Dims)
	for i := 0; i < Dims; i++ {
		if i%2 == 0 {
			v3[i] = 1.0
		} else {
			v3[i] = -1.0
		}
	}
	norm3 := Normalize(v3)
	simOrtho := DotProduct(norm1, norm3)
	if simOrtho > 0.001 || simOrtho < -0.001 {
		t.Fatalf("expected similarity ~0.0 for orthogonal vectors, got %f", simOrtho)
	}
}

func TestProject2D(t *testing.T) {
	v := make([]float32, Dims)
	for i := 0; i < Dims; i++ {
		v[i] = 0.03
	}
	normV := Normalize(v)

	w := make([][]float32, Dims)
	for i := 0; i < Dims; i++ {
		w[i] = []float32{0.01, -0.01}
	}

	x, y, err := Project2D(normV, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if x < -1.0 || x > 1.0 || y < -1.0 || y > 1.0 {
		t.Fatalf("coordinates out of bounds: x=%f, y=%f", x, y)
	}
}

func BenchmarkCosineSimilarity768D(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	v1 := make([]float32, Dims)
	v2 := make([]float32, Dims)

	for i := 0; i < Dims; i++ {
		v1[i] = rng.Float32()
		v2[i] = rng.Float32()
	}

	v1 = Normalize(v1)
	v2 = Normalize(v2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = DotProduct(v1, v2)
	}
}
