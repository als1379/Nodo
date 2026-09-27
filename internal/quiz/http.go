package quiz

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
	mux.Handle("GET /v1/quiz/word", requireAuth(http.HandlerFunc(h.next)))
	mux.Handle("POST /v1/quiz/word/{id}/answer", requireAuth(http.HandlerFunc(h.answer)))
}

func (h *HTTP) next(w http.ResponseWriter, r *http.Request) {
	question, err := h.service.Next(r.Context())
	if err != nil {
		h.logger.Error("question generation failed", "error", err)
		writeError(w, http.StatusBadGateway, "could not generate question")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"question": question})
}

func (h *HTTP) answer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var input struct {
		Option *int `json:"option"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil || input.Option == nil || *input.Option < 0 || *input.Option > 3 {
		writeError(w, http.StatusBadRequest, "option must be an integer from 0 to 3")
		return
	}
	correct, answer, err := h.service.Answer(r.PathValue("id"), *input.Option)
	if errors.Is(err, ErrQuestionNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"correct": correct, "correct_answer": answer})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
