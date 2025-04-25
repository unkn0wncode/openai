// Package embedding provides functions to use OpenAI's Embeddings API.
package embedding

import (
	"math"
)

// Service is the service for the Embeddings API.
type Service interface {
	One(input string) (Vector, error)
	Array(inputs ...string) ([]Vector, error)

	// Dimensions returns the number of dimensions used for embeddings.
	Dimensions() int

	// SetDimensions sets the number of dimensions used for embeddings.
	SetDimensions(int)
}

// Vector is a calculated embedding.
type Vector []float64

// AngleDiff calculates the cosine similarity between two vectors.
// 1 means that vectors are parallel, 0 - orthogonal, -1 - antiparallel.
func (v Vector) AngleDiff(v2 Vector) float64 {
	if len(v) != len(v2) {
		return 0
	}

	var dot, mag1, mag2 float64
	for i := range v {
		dot += (v)[i] * (v2)[i]
		mag1 += (v)[i] * (v)[i]
		mag2 += (v2)[i] * (v2)[i]
	}

	mag1 = math.Sqrt(mag1)
	mag2 = math.Sqrt(mag2)

	return dot / (mag1 * mag2)
}

// Distance calculates the Euclidean distance between two vectors.
func (v Vector) Distance(v2 Vector) float64 {
	if len(v) != len(v2) {
		return 0
	}

	var sum float64
	for i := range v {
		sum += ((v)[i] - (v2)[i]) * ((v)[i] - (v2)[i])
	}

	return math.Sqrt(sum)
}
