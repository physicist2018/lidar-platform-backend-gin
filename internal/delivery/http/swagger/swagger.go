package swagger

import (
	"embed"
	"net/http"
	"strings"

	"github.com/kshmirko/lidar-platform-go/docs"
)

//go:embed index.html
var customUI embed.FS

// NewHandler returns an http.Handler that serves the Swagger UI.
// It serves:
//   - /swagger/ or /swagger/index.html — custom HTML pointing to local spec
//   - /swagger/swagger.json — the embedded OpenAPI spec
func NewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/swagger")
		switch {
		case path == "/swagger.json":
			docs.SwaggerJSONHandler(w, r)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			data, _ := customUI.ReadFile("index.html")
			w.Write(data)
		}
	})
}
