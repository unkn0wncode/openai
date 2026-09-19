// Package models / embeddings.go contains list and properties of OpenAI embedding models.
package models

const (
	DefaultEmbedding    = TextEmbedding3Small
	TextEmbedding3Large = "text-embedding-3-large"
	TextEmbedding3Small = "text-embedding-3-small"
)

// DataEmbedding contains the price in USD per million tokens for each embedding model.
var DataEmbedding = map[string]float64{
	TextEmbedding3Large: 0.13,
	TextEmbedding3Small: 0.02,
}
