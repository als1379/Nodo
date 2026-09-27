package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type HTTP struct{ service *Service }
type contextKey string

const userContextKey contextKey = "authenticated_user"

func NewHTTP(service *Service) *HTTP { return &HTTP{service: service} }

func (h *HTTP) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/register", h.register)
	mux.HandleFunc("POST /v1/auth/login", h.login)
	mux.Handle("POST /v1/auth/logout", h.RequireAuth(http.HandlerFunc(h.logout)))
	mux.Handle("GET /v1/auth/me", h.RequireAuth(http.HandlerFunc(h.me)))
}

func (h *HTTP) register(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	if msg := validateCredentials(input, true); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}
	result, err := h.service.Register(r.Context(), input.Email, input.Password)
	if errors.Is(err, ErrEmailTaken) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *HTTP) login(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	if msg := validateCredentials(input, false); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}
	result, err := h.service.Login(r.Context(), input.login(), input.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *HTTP) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r.Context(), bearerToken(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTP) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user": r.Context().Value(userContextKey)})
}

func (h *HTTP) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, ErrInvalidToken.Error())
			return
		}
		user, err := h.service.Authenticate(r.Context(), token)
		if errors.Is(err, ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, ErrInvalidToken.Error())
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}

type credentials struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var input credentials
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return credentials{}, false
	}
	return input, true
}
func validateCredentials(c credentials, strong bool) string {
	if strong && (!strings.Contains(c.Email, "@") || len(c.Email) > 254) {
		return "a valid email is required"
	}
	if !strong && c.login() == "" {
		return "login is required"
	}
	if strong && (len(c.Password) < 10 || len(c.Password) > 72) {
		return "password must be between 10 and 72 characters"
	}
	if !strong && c.Password == "" {
		return "password is required"
	}
	return ""
}
func (c credentials) login() string {
	if strings.TrimSpace(c.Login) != "" {
		return c.Login
	}
	return c.Email
}
func bearerToken(r *http.Request) string {
	p := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(p) == 2 && strings.EqualFold(p[0], "Bearer") {
		return strings.TrimSpace(p[1])
	}
	return ""
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
