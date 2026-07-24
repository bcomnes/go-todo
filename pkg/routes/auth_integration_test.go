package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bcomnes/go-todo/pkg/database"
	"github.com/bcomnes/go-todo/pkg/httpapi"
	"github.com/bcomnes/go-todo/pkg/models"
	"github.com/bcomnes/go-todo/pkg/security"
)

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func TestAuthLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	db, err := database.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	api, err := httpapi.New(db, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	handler := api.Handler()

	suffix := time.Now().UnixNano()
	username := fmt.Sprintf("user%d", suffix)
	email := fmt.Sprintf("user%d@example.test", suffix)
	password := "correct horse battery staple"
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM public.users WHERE email = $1`, email)
	})

	registerBody, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	register := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(registerBody))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, register)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", registerResponse.Code, registerResponse.Body.String())
	}
	if len(registerResponse.Result().Cookies()) != 0 {
		t.Fatal("JSON registration unexpectedly set a browser session cookie")
	}
	var registrationAuth authResponse
	if err := json.NewDecoder(registerResponse.Body).Decode(&registrationAuth); err != nil {
		t.Fatalf("decode registration response: %v", err)
	}
	if registrationAuth.Token == "" || registrationAuth.User.Email != email {
		t.Fatalf("registration auth response = %#v", registrationAuth)
	}
	registrationAccount := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	registrationAccount.Header.Set("Authorization", "Bearer "+registrationAuth.Token)
	registrationAccountResponse := httptest.NewRecorder()
	handler.ServeHTTP(registrationAccountResponse, registrationAccount)
	if registrationAccountResponse.Code != http.StatusOK {
		t.Fatalf("account with registration token status = %d, body = %s", registrationAccountResponse.Code, registrationAccountResponse.Body.String())
	}

	var passwordHash string
	var passwordMatches bool
	if err := db.QueryRow(`
		SELECT password_hash, password_hash = public.crypt($2, password_hash)
		FROM public.users
		WHERE email = $1
	`, email, password).Scan(&passwordHash, &passwordMatches); err != nil {
		t.Fatalf("read password hash: %v", err)
	}
	if passwordHash == password {
		t.Fatal("database stored the plaintext password")
	}
	if !passwordMatches {
		t.Fatal("database password verification failed")
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	login := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(loginBody))
	login.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResponse.Code, loginResponse.Body.String())
	}
	if len(loginResponse.Result().Cookies()) != 0 {
		t.Fatal("JSON login unexpectedly set a browser session cookie")
	}
	if cacheControl := loginResponse.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("JSON login cache control = %q, want no-store", cacheControl)
	}

	var auth authResponse
	if err := json.NewDecoder(loginResponse.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	parsedToken, err := security.ParseToken(auth.Token)
	if err != nil {
		t.Fatalf("parse returned token: %v", err)
	}

	var tokenHash []byte
	var tokenMatches bool
	if err := db.QueryRow(`
			SELECT token_hash, token_hash = public.digest($2, 'sha256')
			FROM public.auth_tokens
			WHERE id = $1
		`, parsedToken.ID, parsedToken.Secret).Scan(&tokenHash, &tokenMatches); err != nil {
		t.Fatalf("read token hash: %v", err)
	}
	if string(tokenHash) == auth.Token || string(tokenHash) == parsedToken.Secret {
		t.Fatal("database stored plaintext token material")
	}
	if len(tokenHash) != 32 {
		t.Fatalf("token digest length = %d, want 32", len(tokenHash))
	}
	if !tokenMatches {
		t.Fatal("database token verification failed")
	}

	account := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	account.Header.Set("Authorization", "Bearer "+auth.Token)
	accountResponse := httptest.NewRecorder()
	handler.ServeHTTP(accountResponse, account)
	if accountResponse.Code != http.StatusOK {
		t.Fatalf("account status = %d, body = %s", accountResponse.Code, accountResponse.Body.String())
	}
	if cacheControl := accountResponse.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("account cache control = %q, want no-store", cacheControl)
	}

	logout := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	logout.Header.Set("Authorization", "Bearer "+auth.Token)
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", logoutResponse.Code, logoutResponse.Body.String())
	}

	accountAfterLogout := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	accountAfterLogout.Header.Set("Authorization", "Bearer "+auth.Token)
	accountAfterLogoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(accountAfterLogoutResponse, accountAfterLogout)
	if accountAfterLogoutResponse.Code != http.StatusUnauthorized {
		t.Fatalf("account after logout status = %d, want %d", accountAfterLogoutResponse.Code, http.StatusUnauthorized)
	}

	form := url.Values{"email": {email}, "password": {password}}
	browserLogin := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	browserLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	browserLogin.Header.Set("Origin", "https://example.com")
	browserLogin.Header.Set("HX-Request", "true")
	browserLoginResponse := httptest.NewRecorder()
	handler.ServeHTTP(browserLoginResponse, browserLogin)
	if browserLoginResponse.Code != http.StatusNoContent {
		t.Fatalf("browser login status = %d, body = %s", browserLoginResponse.Code, browserLoginResponse.Body.String())
	}
	if redirect := browserLoginResponse.Header().Get("HX-Redirect"); redirect != "/todos" {
		t.Fatalf("browser login redirect = %q, want /todos", redirect)
	}
	cookies := browserLoginResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("browser login cookie count = %d, want 1", len(cookies))
	}
	sessionCookie := cookies[0]
	if !sessionCookie.Secure || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteLaxMode || sessionCookie.Value == "" {
		t.Fatalf("browser session cookie has unsafe attributes: %#v", sessionCookie)
	}

	cookieOnlyAPIRequest := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	cookieOnlyAPIRequest.AddCookie(sessionCookie)
	cookieOnlyAPIResponse := httptest.NewRecorder()
	handler.ServeHTTP(cookieOnlyAPIResponse, cookieOnlyAPIRequest)
	if cookieOnlyAPIResponse.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-only API status = %d, want %d", cookieOnlyAPIResponse.Code, http.StatusUnauthorized)
	}

	accountPage := httptest.NewRequest(http.MethodGet, "/account", nil)
	accountPage.AddCookie(sessionCookie)
	accountPageResponse := httptest.NewRecorder()
	handler.ServeHTTP(accountPageResponse, accountPage)
	if accountPageResponse.Code != http.StatusOK {
		t.Fatalf("account page status = %d, body = %s", accountPageResponse.Code, accountPageResponse.Body.String())
	}
	if !strings.Contains(accountPageResponse.Body.String(), username) || !strings.Contains(accountPageResponse.Body.String(), email) {
		t.Fatal("account page does not contain the authenticated user")
	}

	accountFragment := httptest.NewRequest(http.MethodGet, "/account?fragment=account-card", nil)
	accountFragment.AddCookie(sessionCookie)
	accountFragment.Header.Set("HX-Request", "true")
	accountFragmentResponse := httptest.NewRecorder()
	handler.ServeHTTP(accountFragmentResponse, accountFragment)
	if accountFragmentResponse.Code != http.StatusOK {
		t.Fatalf("account fragment status = %d, body = %s", accountFragmentResponse.Code, accountFragmentResponse.Body.String())
	}
	if strings.Contains(accountFragmentResponse.Body.String(), "<!doctype html>") {
		t.Fatal("account fragment unexpectedly contains the shared layout")
	}

	browserLogout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	browserLogout.AddCookie(sessionCookie)
	browserLogout.Header.Set("Origin", "https://example.com")
	browserLogout.Header.Set("HX-Request", "true")
	browserLogoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(browserLogoutResponse, browserLogout)
	if browserLogoutResponse.Code != http.StatusNoContent {
		t.Fatalf("browser logout status = %d, body = %s", browserLogoutResponse.Code, browserLogoutResponse.Body.String())
	}
	if redirect := browserLogoutResponse.Header().Get("HX-Redirect"); redirect != "/" {
		t.Fatalf("browser logout redirect = %q, want /", redirect)
	}

	accountAfterBrowserLogout := httptest.NewRequest(http.MethodGet, "/account", nil)
	accountAfterBrowserLogout.AddCookie(sessionCookie)
	accountAfterBrowserLogout.Header.Set("HX-Request", "true")
	accountAfterBrowserLogoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(accountAfterBrowserLogoutResponse, accountAfterBrowserLogout)
	if accountAfterBrowserLogoutResponse.Code != http.StatusNoContent {
		t.Fatalf("account after browser logout status = %d, want %d", accountAfterBrowserLogoutResponse.Code, http.StatusNoContent)
	}
	if redirect := accountAfterBrowserLogoutResponse.Header().Get("HX-Redirect"); redirect != "/login" {
		t.Fatalf("expired browser session redirect = %q, want /login", redirect)
	}
}

func TestBrowserRegistrationStartsSession(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	db, err := database.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	api, err := httpapi.New(db, time.Hour)
	if err != nil {
		t.Fatalf("initialize API: %v", err)
	}
	handler := api.Handler()

	suffix := time.Now().UnixNano()
	username := fmt.Sprintf("browser%d", suffix)
	email := fmt.Sprintf("browser%d@example.test", suffix)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM public.users WHERE email = $1`, email)
	})

	form := url.Values{
		"username": {username},
		"email":    {email},
		"password": {"correct horse battery staple"},
	}
	register := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	register.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	register.Header.Set("Origin", "https://example.com")
	register.Header.Set("HX-Request", "true")
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, register)
	if registerResponse.Code != http.StatusNoContent {
		t.Fatalf("browser registration status = %d, body = %s", registerResponse.Code, registerResponse.Body.String())
	}
	if redirect := registerResponse.Header().Get("HX-Redirect"); redirect != "/todos" {
		t.Fatalf("browser registration redirect = %q, want /todos", redirect)
	}
	cookies := registerResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("browser registration cookie count = %d, want 1", len(cookies))
	}
	sessionCookie := cookies[0]
	if !sessionCookie.Secure || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteLaxMode || sessionCookie.Value == "" {
		t.Fatalf("browser registration cookie has unsafe attributes: %#v", sessionCookie)
	}

	todosPage := httptest.NewRequest(http.MethodGet, "/todos", nil)
	todosPage.AddCookie(sessionCookie)
	todosPageResponse := httptest.NewRecorder()
	handler.ServeHTTP(todosPageResponse, todosPage)
	if todosPageResponse.Code != http.StatusOK {
		t.Fatalf("todos after registration status = %d, body = %s", todosPageResponse.Code, todosPageResponse.Body.String())
	}
	if !strings.Contains(todosPageResponse.Body.String(), username) {
		t.Fatal("todos page does not show the newly registered session")
	}
}
