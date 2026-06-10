package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, statusCode int, msg string) {
	JSON(w, statusCode, map[string]string{"error": msg})
}

// BindAndValidate decodes JSON body and validates it.
func BindAndValidate(r *http.Request, validate *validator.Validate, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return errors.New("invalid request body")
	}
	if err := validate.Struct(target); err != nil {
		return err
	}
	return nil
}
