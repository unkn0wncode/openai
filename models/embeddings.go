// Package models / embeddings.go contains list and properties of OpenAI embedding models.
package models

const (
	DefaultEmbedding = ThreeSmall
	ThreeLarge       = "text-embedding-3-large"
	ThreeSmall       = "text-embedding-3-small"
)

// DataEmbedding contains price per 1 token for each embedding model.
var DataEmbedding = map[string]float64{
	ThreeLarge: 0.00000013,
	ThreeSmall: 0.00000002,
}
