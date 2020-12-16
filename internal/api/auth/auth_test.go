package auth

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_NoTokenShouldReturnUnauthorized(t *testing.T) {
	e := echo.New()
	m := (&Middleware{signingKey: []byte("test")}).Middleware()(okHandler())

	// no token shouldn't be able to access
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := m(c); err != ErrInvalidOrMissingAuth {
		t.Errorf("ErrInvalidOrMissingAuth was expected")
	}
}

func TestMiddleware_InvalidTokenShouldReturnUnauthorized(t *testing.T) {
	e := echo.New()
	m := (&Middleware{signingKey: []byte("test")}).Middleware()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Add(echo.HeaderAuthorization, "zarBR")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := m(okHandler())(c); err != ErrInvalidOrMissingAuth {
		t.Errorf("ErrInvalidOrMissingAuth was expected")
	}
}

func TestMiddleware_BadRightsShouldReturnUnauthorized(t *testing.T) {
	e := echo.New()
	m := (&Middleware{signingKey: []byte("test")}).Middleware()

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	req.Header.Add(echo.HeaderAuthorization, "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6IkpvaG4gRG9lIiwicmlnaHRzIjp7IkdFVCI6WyIvdXNlcnMiXSwiUE9TVCI6WyIvc2VhcmNoIl19fQ.fRx0Q66ZgnY_rKCf-9Vaz6gzGKH_tKSgkVHhoQMtKfM")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := m(okHandler())(c); err != ErrAccessUnauthorized {
		t.Errorf("ErrAccessUnauthorized was expected")
	}
}

func TestMiddleware(t *testing.T) {
	e := echo.New()
	m := (&Middleware{signingKey: []byte("test")}).Middleware()

	req := httptest.NewRequest(http.MethodGet, "/users?id=10", nil)
	req.Header.Add(echo.HeaderAuthorization, "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6IkpvaG4gRG9lIiwicmlnaHRzIjp7IkdFVCI6WyIvdXNlcnMiXSwiUE9TVCI6WyIvc2VhcmNoIl19fQ.fRx0Q66ZgnY_rKCf-9Vaz6gzGKH_tKSgkVHhoQMtKfM")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = m(okHandler())(c)
	if rec.Code != http.StatusOK {
		t.Fail()
	}
	b, err := ioutil.ReadAll(rec.Body)
	if err != nil {
		t.Fail()
	}
	if string(b) != "Hello, John Doe" {
		t.Fail()
	}

	if token := c.Get("username").(string); token != "John Doe" {
		t.Fail()
	}
}

func okHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		if username := c.Get("username").(string); username != "" {
			return c.String(http.StatusOK, fmt.Sprintf("Hello, %s", username))
		}

		return c.NoContent(http.StatusNoContent)
	}
}
