package math

import (
	"fmt"
	"math"
)

// Dims is the standard dimension for Gemini and contemporary embeddings.
const Dims = 768

// DotProduct computes the dot product of two float32 slices.
// When both slices are unit-normalized, the dot product equals cosine similarity.
// We unroll the loop by 8 for SIMD-friendly compiler autovectorization.
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32 = 0
	n := len(a)
	i := 0

	// Unroll 8x for compiler vectorization
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

	// Remainder
	for ; i < n; i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// Normalize normalizes a vector to have unit Euclidean norm (L2 norm = 1.0).
func Normalize(v []float32) []float32 {
	var sumSq float64 = 0
	for _, val := range v {
		sumSq += float64(val) * float64(val)
	}

	norm := float32(math.Sqrt(sumSq))
	if norm == 0 {
		return v
	}

	normalized := make([]float32, len(v))
	invNorm := 1.0 / norm
	for i, val := range v {
		normalized[i] = val * invNorm
	}

	return normalized
}

// Project2D projects a 768D unit vector into 2D coordinates [x, y] in [-1.0, 1.0]
// using the precomputed static projection matrix W of size (Dims x 2).
func Project2D(v []float32, w [][]float32) (float32, float32, error) {
	if len(v) != Dims {
		return 0, 0, fmt.Errorf("vector dim mismatch: expected %d, got %d", Dims, len(v))
	}
	if len(w) != Dims {
		return 0, 0, fmt.Errorf("matrix row mismatch: expected %d, got %d", Dims, len(w))
	}

	var x float32 = 0
	var y float32 = 0

	for i := 0; i < Dims; i++ {
		val := v[i]
		x += val * w[i][0]
		y += val * w[i][1]
	}

	// Clamp to [-1.0, 1.0]
	if x > 1.0 {
		x = 1.0
	} else if x < -1.0 {
		x = -1.0
	}

	if y > 1.0 {
		y = 1.0
	} else if y < -1.0 {
		y = -1.0
	}

	return x, y, nil
}
