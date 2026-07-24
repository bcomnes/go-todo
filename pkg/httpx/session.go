package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bcomnes/go-todo/pkg/auth"
	"github.com/bcomnes/go-todo/pkg/models"
)

const sessionCookieName = "go_todo_session"

type sessionContextKey struct{}

// Sessions adapts the auth service to bearer headers, browser cookies,
// authentication middleware, and same-origin browser action checks.
type Sessions struct {
	auth          *auth.Service
	secureCookies bool
}

// NewSessions creates an HTTP session manager. secureCookies should be false
// only for explicitly configured local plain-HTTP development.
func NewSessions(authService *auth.Service, secureCookies bool) (*Sessions, error) {
	if authService == nil {
		return nil, errors.New("authentication service is required")
	}
	return &Sessions{auth: authService, secureCookies: secureCookies}, nil
}

// OptionalUser returns the cookie-authenticated user when available. Anonymous
// and unavailable authentication both produce nil because this method is for
// public presentation only, never authorization.
func (sessions *Sessions) OptionalUser(r *http.Request) *models.User {
	session, err := sessions.authenticate(r, cookieToken)
	if err != nil {
		return nil
	}
	return &session.User
}

// RequireAPI authenticates a JSON route with a bearer token and attaches its
// session to context. Browser cookies are deliberately not API credentials.
func (sessions *Sessions) RequireAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := sessions.authenticate(r, bearerToken)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				WriteError(w, http.StatusUnauthorized, auth.ErrUnauthorized.Error())
				return
			}
			WriteError(w, http.StatusServiceUnavailable, auth.ErrUnavailable.Error())
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey{}, session)
		next(w, r.WithContext(ctx))
	}
}

// RequirePage authenticates a browser route with its session cookie, clearing
// and redirecting invalid sessions to login while treating service failures as
// unavailable. Authorization headers are deliberately not browser sessions.
func (sessions *Sessions) RequirePage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		session, err := sessions.authenticate(r, cookieToken)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				sessions.ClearCookie(w)
				Redirect(w, r, "/login")
				return
			}
			http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey{}, session)
		next(w, r.WithContext(ctx))
	}
}

// Current returns the session installed by RequireAPI or RequirePage.
func (sessions *Sessions) Current(ctx context.Context) (auth.Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(auth.Session)
	return session, ok
}

// SetCookie stores an opaque token in the hardened browser session cookie.
func (sessions *Sessions) SetCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   sessions.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie expires the browser session cookie with the same security attributes.
func (sessions *Sessions) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sessions.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// HasSameOrigin validates the Origin or Referer of a browser state-changing request.
func (sessions *Sessions) HasSameOrigin(r *http.Request) bool {
	source := r.Header.Get("Origin")
	if source == "" {
		source = r.Header.Get("Referer")
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	expectedScheme := "http"
	if sessions.secureCookies || r.TLS != nil {
		expectedScheme = "https"
	}
	return parsed.Scheme == expectedScheme && strings.EqualFold(parsed.Host, r.Host)
}

type tokenExtractor func(*http.Request) (string, error)

func (sessions *Sessions) authenticate(r *http.Request, extract tokenExtractor) (auth.Session, error) {
	plaintext, err := extract(r)
	if err != nil {
		return auth.Session{}, err
	}
	return sessions.auth.Authenticate(r.Context(), plaintext)
}

func bearerToken(r *http.Request) (string, error) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", auth.ErrUnauthorized
	}
	return parts[1], nil
}

func cookieToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", auth.ErrUnauthorized
	}
	return cookie.Value, nil
}
