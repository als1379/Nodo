package dictionary

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type HTTP struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTP(service *Service, logger *slog.Logger) *HTTP {
	return &HTTP{service: service, logger: logger}
}

func (h *HTTP) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /v1/dictionary/{word}", requireAuth(http.HandlerFunc(h.lookup)))
}

func (h *HTTP) lookup(w http.ResponseWriter, r *http.Request) {
	var result LookupResult
	var err error
	switch r.URL.Query().Get("view") {
	case "preview":
		result, err = h.service.Preview(r.Context(), r.PathValue("word"))
	case "entry":
		result, err = h.service.Entry(r.Context(), r.PathValue("word"))
	case "":
		result, err = h.service.Lookup(r.Context(), r.PathValue("word"))
	default:
		writeError(w, 400, "view must be preview or entry")
		return
	}
	if errors.Is(err, ErrInvalidWord) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		h.logger.Error("word lookup failed", "word", r.PathValue("word"), "error", err)
		if errors.Is(err, ErrWordNotFound) {
			writeError(w, 404, "Italian word not found")
		} else {
			writeError(w, 502, "The dictionary is temporarily unavailable. Please try again.")
		}
		return
	}
	if result.LessonWarning != nil {
		h.logger.Error("learning agent failed", "word", result.Details.Word, "error", result.LessonWarning)
	}
	writeJSON(w, http.StatusOK, lookupResponse{Details: result.Details})
}

type lookupResponse struct {
	Details WordDetails `json:"details"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
