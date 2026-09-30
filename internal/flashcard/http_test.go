package flashcard

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidRequestsNeverReachStore(t *testing.T) {
	mux := http.NewServeMux()
	NewHTTP(nil, slog.New(slog.NewTextHandler(io.Discard, nil))).RegisterRoutes(mux, func(h http.Handler) http.Handler { return h })
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/v1/flashcards/sessions", `{"mode":"invalid"}`},
		{"POST", "/v1/flashcards/sessions", `{"level":"A1"}`},
		{"POST", "/v1/flashcards/sessions", `{} {}`},
		{"POST", "/v1/flashcards/sessions/not-a-uuid/answers", `{"word_id":1,"known":true}`},
		{"POST", "/v1/flashcards/sessions/not-a-uuid/answers", `{"word_id":1}`},
		{"PUT", "/v1/flashcards/settings", `{}`},
		{"PUT", "/v1/flashcards/settings", `{"difficulty_target":101}`},
		{"PUT", "/v1/flashcards/settings", `{"difficulty_target":-1}`},
		{"GET", "/v1/flashcards/words?status=unknown", ``},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status %d: %s", res.Code, res.Body.String())
			}
		})
	}
}
