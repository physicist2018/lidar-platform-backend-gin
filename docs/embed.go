package docs

import (
	"embed"
	"io"
	"net/http"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/response"
)

// SwaggerFS embeds swagger.json and swagger.yaml for serving.
//
//go:embed swagger.json swagger.yaml
var SwaggerFS embed.FS

func init() {
	// Verify files are embedded at build time
	if _, err := SwaggerFS.ReadFile("swagger.json"); err != nil {
		panic("swagger.json not embedded: " + err.Error())
	}
	if _, err := SwaggerFS.ReadFile("swagger.yaml"); err != nil {
		panic("swagger.yaml not embedded: " + err.Error())
	}
}

// SwaggerJSONHandler serves swagger.json from embedded FS.
func SwaggerJSONHandler(w http.ResponseWriter, r *http.Request) {
	f, err := SwaggerFS.Open("swagger.json")
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]string{"error": "swagger.json not found"})
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, f)
}

// SwaggerYAMLHandler serves swagger.yaml from embedded FS.
func SwaggerYAMLHandler(w http.ResponseWriter, r *http.Request) {
	f, err := SwaggerFS.Open("swagger.yaml")
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]string{"error": "swagger.yaml not found"})
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/yaml")
	io.Copy(w, f)
}
