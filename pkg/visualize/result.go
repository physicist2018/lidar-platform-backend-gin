package visualize

// Result holds the generated visualization data.
// It mirrors usecase.VisualizeResult without importing internal.
type Result struct {
	ContentType string
	Body        []byte
}
