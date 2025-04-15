package embedding

// DefaultModel is the model that's used when not specified.
// Can be altered externally.
var DefaultModel = Model3Small

const (
	ModelAda2    = "text-embedding-ada-002"
	Model3Large  = "text-embedding-3-large"
	Model3Small  = "text-embedding-3-small"
)
