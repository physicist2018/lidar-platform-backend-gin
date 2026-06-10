package docs

import (
	"embed"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
)

// SwaggerFS embeds swagger.json and swagger.yaml for serving by Echo.
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

// SwaggerJSONHandler serves swagger.json from embedded FS as an Echo handler.
func SwaggerJSONHandler(c *echo.Context) error {
	f, err := SwaggerFS.Open("swagger.json")
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "swagger.json not found"})
	}
	defer f.Close()
	c.Response().Header().Set("Content-Type", "application/json")
	_, err = io.Copy(c.Response(), f)
	return err
}

// SwaggerYAMLHandler serves swagger.yaml from embedded FS as an Echo handler.
func SwaggerYAMLHandler(c *echo.Context) error {
	f, err := SwaggerFS.Open("swagger.yaml")
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "swagger.yaml not found"})
	}
	defer f.Close()
	c.Response().Header().Set("Content-Type", "text/yaml")
	_, err = io.Copy(c.Response(), f)
	return err
}
