package auth

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

type key int

const (
	usernameKey key = iota
)

// Token is the authentication token used by processes when dialing with the API
type Token struct {
	// Username used for logging purposes
	Username string `json:"username"`

	// Rights that the token provides
	// Format is: METHOD - list of paths
	Rights map[string][]string `json:"rights"`
}

// Middleware is the authentication middleware
type Middleware struct {
	signingKey []byte
}

// NewMiddleware create a new Middleware instance with given secret token signing key
func NewMiddleware(signingKey []byte) *Middleware {
	return &Middleware{signingKey: signingKey}
}

// Middleware return an net/http compatible middleware func to use
func (m *Middleware) Middleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract authorization header
			tokenStr := r.Header.Get("Authorization")
			if tokenStr == "" {
				log.Warn().Msg("missing token")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

			// Decode the JWT token
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				// Validate expected alg
				if v, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || v.Name != "HS256" {
					return nil, fmt.Errorf("unexpected signing method: %s", t.Header["alg"])
				}

				// Return signing secret
				return m.signingKey, nil
			})
			if err != nil {
				log.Err(err).Msg("error while decoding JWT token")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// From here we have a valid JWT token, extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.Err(err).Msg("error while decoding token claims")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rights := map[string][]string{}
			for method, paths := range claims["rights"].(map[string]interface{}) {
				for _, path := range paths.([]interface{}) {
					rights[method] = append(rights[method], path.(string))
				}
			}

			t := Token{
				Username: claims["username"].(string),
				Rights:   rights,
			}

			// Validate rights
			paths, contains := t.Rights[r.Method]
			if !contains {
				log.Warn().
					Str("username", t.Username).
					Str("method", r.Method).
					Str("resource", r.URL.Path).
					Msg("Access to resources is unauthorized")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			authorized := false
			for _, path := range paths {
				if path == r.URL.Path {
					authorized = true
					break
				}
			}

			if !authorized {
				log.Warn().
					Str("username", t.Username).
					Str("method", r.Method).
					Str("resource", r.URL.Path).
					Msg("Access to resources is unauthorized")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Everything's fine, call next handler ;D
			ctx := context.WithValue(r.Context(), usernameKey, t.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
