package auth

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_NoTokenShouldReturnUnauthorized(t *testing.T) {
	m := (&Middleware{signingKey: []byte("test")}).Middleware()(okHandler())

	// no token shouldn't be able to access
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusUnauthorized was expected")
	}
}

func TestMiddleware_InvalidTokenShouldReturnUnauthorized(t *testing.T) {
	m := (&Middleware{signingKey: []byte("test")}).Middleware()(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Add("Authorization", "zarBR")
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusUnauthorized was expected")
	}
}

func TestMiddleware_BadRightsShouldReturnUnauthorized(t *testing.T) {
	m := (&Middleware{signingKey: []byte("test")}).Middleware()(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6IkpvaG4gRG9lIiwicmlnaHRzIjp7IkdFVCI6WyIvdXNlcnMiXSwiUE9TVCI6WyIvc2VhcmNoIl19fQ.fRx0Q66ZgnY_rKCf-9Vaz6gzGKH_tKSgkVHhoQMtKfM")
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusUnauthorized was expected")
	}
}

func TestMiddleware(t *testing.T) {
	m := (&Middleware{signingKey: []byte("test")}).Middleware()(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/users?id=10", nil)
	req.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6IkpvaG4gRG9lIiwicmlnaHRzIjp7IkdFVCI6WyIvdXNlcnMiXSwiUE9TVCI6WyIvc2VhcmNoIl19fQ.fRx0Q66ZgnY_rKCf-9Vaz6gzGKH_tKSgkVHhoQMtKfM")
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("StatusUnauthorized was expected")
	}

	b, err := ioutil.ReadAll(rec.Body)
	if err != nil {
		t.Fail()
	}
	if string(b) != "Hello, John Doe" {
		t.Fail()
	}
}

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if username := r.Context().Value(usernameKey).(string); username != "" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf("Hello, %s", username)))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
