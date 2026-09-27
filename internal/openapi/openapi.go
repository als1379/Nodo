package openapi

import (
	"embed"
	"net/http"
)

//go:embed openapi.json
var files embed.FS

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spec, err := files.ReadFile("openapi.json")
		if err != nil { http.Error(w, "OpenAPI specification unavailable", http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(spec)
	})
}
