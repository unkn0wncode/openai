// Package models / embeddings.go contains list and properties of OpenAI embedding models.
package models

const (
	DefaultEmbedding = ThreeSmall
	Ada2             = "text-embedding-ada-002"
	ThreeLarge       = "text-embedding-3-large"
	ThreeSmall       = "text-embedding-3-small"
)

// DataEmbedding contains price per 1 token for each embedding model.
var DataEmbedding = map[string]float64{
	Ada2:       0.00000010,
	ThreeLarge: 0.00000013,
	ThreeSmall: 0.00000002,
}
