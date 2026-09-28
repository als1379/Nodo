package dictionary

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"
)

type HTTP struct {
	dictionary *Dictionary
	logger     *slog.Logger
}

func NewHTTP(dictionary *Dictionary, logger *slog.Logger) *HTTP {
	return &HTTP{dictionary: dictionary, logger: logger}
}

func (h *HTTP) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /v1/dictionary/{word}", requireAuth(http.HandlerFunc(h.lookup)))
}

func (h *HTTP) lookup(w http.ResponseWriter, r *http.Request) {
	word := strings.ToLower(strings.TrimSpace(r.PathValue("word")))
	if word == "" || utf8.RuneCountInString(word) > 80 {
		writeError(w, http.StatusBadRequest, "word must be between 1 and 80 characters")
		return
	}
	details, err := h.dictionary.Lookup(r.Context(), word)
	if err != nil {
		h.logger.Error("word lookup failed", "word", word, "error", err)
		writeError(w, http.StatusNotFound, "Italian word not found")
		return
	}
	if details.LearningFailure != nil {
		h.logger.Error("learning agent failed", "word", word, "error", details.LearningFailure)
	}
	writeJSON(w, http.StatusOK, map[string]any{"details": details})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
