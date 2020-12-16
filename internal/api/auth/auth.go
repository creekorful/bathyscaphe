package auth

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

// ErrInvalidOrMissingAuth is returned if the authorization header is absent or invalid
var ErrInvalidOrMissingAuth = &echo.HTTPError{
	Code:    http.StatusUnauthorized,
	Message: "Invalid or missing `Authorization` header",
}

// ErrAccessUnauthorized is returned if the token doesn't grant access to the current resource
var ErrAccessUnauthorized = &echo.HTTPError{
	Code:    http.StatusUnauthorized,
	Message: "Access to the resource is not authorized",
}

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

// Middleware return an echo compatible middleware func to use
func (m *Middleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract authorization header
			tokenStr := c.Request().Header.Get(echo.HeaderAuthorization)
			if tokenStr == "" {
				return ErrInvalidOrMissingAuth
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
				return ErrInvalidOrMissingAuth
			}

			// From here we have a valid JWT token, extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return fmt.Errorf("error while parsing token claims")
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
			paths, contains := t.Rights[c.Request().Method]
			if !contains {
				return ErrAccessUnauthorized
			}

			authorized := false
			for _, path := range paths {
				if path == c.Request().URL.Path {
					authorized = true
					break
				}
			}

			if !authorized {
				return ErrAccessUnauthorized
			}

			// Set user context
			c.Set("username", t.Username)

			// Everything's fine, call next handler ;D
			return next(c)
		}
	}
}
