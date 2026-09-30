package flashcard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"

	"nodo/internal/auth"

	"github.com/jackc/pgx/v5/pgtype"
)

type StudyStore interface {
	Overview(context.Context, string) (Overview, error)
	SaveSettings(context.Context, string, Settings) error
	CreateSession(context.Context, string, SessionMode) (Session, error)
	Answer(context.Context, string, string, int64, bool) (WordProgress, error)
	ListProgress(context.Context, string, bool) ([]WordProgress, error)
}

type HTTP struct {
	study  StudyStore
	logger *slog.Logger
}

func NewHTTP(study StudyStore, logger *slog.Logger) *HTTP {
	return &HTTP{study: study, logger: logger}
}

func (h *HTTP) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("POST /v1/flashcards/sessions", requireAuth(http.HandlerFunc(h.createSession)))
	mux.Handle("POST /v1/flashcards/sessions/{session}/answers", requireAuth(http.HandlerFunc(h.answer)))
	mux.Handle("GET /v1/flashcards/overview", requireAuth(http.HandlerFunc(h.overview)))
	mux.Handle("PUT /v1/flashcards/settings", requireAuth(http.HandlerFunc(h.settings)))
	mux.Handle("GET /v1/flashcards/words", requireAuth(http.HandlerFunc(h.words)))
}

func (h *HTTP) createSession(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	var input struct {
		Mode string `json:"mode"`
	}
	if err := decodeInput(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	mode, err := ParseSessionMode(input.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	session, err := h.study.CreateSession(r.Context(), user.ID, mode)
	if err != nil {
		h.logger.Error("create study session failed", "error", err)
		if errors.Is(err, ErrNoReviewWords) || errors.Is(err, ErrNoWords) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not start lesson")
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (h *HTTP) answer(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	var input struct {
		WordID int64 `json:"word_id"`
		Known  *bool `json:"known"`
	}
	if err := decodeInput(w, r, &input); err != nil || input.WordID < 1 || input.Known == nil {
		writeError(w, http.StatusBadRequest, "word_id and known are required")
		return
	}
	var sessionUUID pgtype.UUID
	if err := sessionUUID.Scan(r.PathValue("session")); err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}
	progress, err := h.study.Answer(r.Context(), user.ID, r.PathValue("session"), input.WordID, *input.Known)
	if errors.Is(err, ErrSessionNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrAlreadyAnswered) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		h.logger.Error("save flashcard answer failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not save answer")
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (h *HTTP) words(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	status := strings.ToLower(r.URL.Query().Get("status"))
	if status != "learning" && status != "learned" {
		writeError(w, http.StatusBadRequest, "status must be learning or learned")
		return
	}
	words, err := h.study.ListProgress(r.Context(), user.ID, status == "learned")
	if err != nil {
		h.logger.Error("list learned words failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not load words")
		return
	}
	if words == nil {
		words = []WordProgress{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "words": words})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeInput(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON value")
	}
	return nil
}
func (h *HTTP) overview(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	out, err := h.study.Overview(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("study overview failed", "error", err)
		writeError(w, 500, "could not load progress")
		return
	}
	writeJSON(w, 200, out)
}
func (h *HTTP) settings(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	var input struct {
		DifficultyTarget *float64 `json:"difficulty_target"`
	}
	if err := decodeInput(w, r, &input); err != nil || input.DifficultyTarget == nil {
		writeError(w, 400, "difficulty_target is required")
		return
	}
	target := *input.DifficultyTarget
	if math.IsNaN(target) || math.IsInf(target, 0) || target < 0 || target > 100 {
		writeError(w, 400, "difficulty_target must be between 0 and 100")
		return
	}
	settings := Settings{DifficultyTarget: target}
	if err := h.study.SaveSettings(r.Context(), user.ID, settings); err != nil {
		writeError(w, 500, "could not save settings")
		return
	}
	writeJSON(w, 200, settings)
}
