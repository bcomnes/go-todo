// Package httpx contains the HTTP transport helpers shared by feature route
// packages. It owns encoding, HTMX redirects, form limits, and page rendering,
// but no feature-specific validation or database operations.
package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/bcomnes/go-todo/pkg/web"
)

const maxRequestBodyBytes = 1 << 20

var errUnsupportedMediaType = errors.New("content type must be application/json")

// DecodeJSON decodes exactly one JSON value, rejects unknown fields, and limits
// the request body to one MiB.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMediaType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return err
	}
	return nil
}

// ParseForm parses a bounded browser form body.
func ParseForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	return r.ParseForm()
}

// WriteJSON writes a non-cacheable JSON response with an explicit status.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// Headers have already been sent; the server logger records connection-level errors.
		return
	}
}

// WriteError writes the shared JSON error envelope.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// WriteDecodeError maps a DecodeJSON error to its public HTTP response.
func WriteDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnsupportedMediaType) {
		WriteError(w, http.StatusUnsupportedMediaType, errUnsupportedMediaType.Error())
		return
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %s", err))
}

// RenderPage renders a complete page before writing response headers.
func RenderPage(w http.ResponseWriter, status int, page *web.Page, data any) {
	renderHTML(w, status, func(buffer *bytes.Buffer) error {
		return page.RenderPage(buffer, data)
	})
}

// RenderFragment renders an allow-listed page fragment before writing headers.
func RenderFragment(w http.ResponseWriter, status int, page *web.Page, fragment string, data any) {
	renderHTML(w, status, func(buffer *bytes.Buffer) error {
		return page.RenderFragment(buffer, fragment, data)
	})
}

// RenderFormError returns only the form fragment for HTMX requests and the full
// page for ordinary browser submissions while preserving the error status.
func RenderFormError(w http.ResponseWriter, r *http.Request, status int, page *web.Page, fragment string, data any) {
	if IsHTMX(r) {
		RenderFragment(w, status, page, fragment, data)
		return
	}
	RenderPage(w, status, page, data)
}

// Redirect returns an HX-Redirect for HTMX and a 303 redirect otherwise.
func Redirect(w http.ResponseWriter, r *http.Request, location string) {
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

// IsHTMX reports whether the request identifies itself as an HTMX request.
func IsHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

func renderHTML(w http.ResponseWriter, status int, render func(*bytes.Buffer) error) {
	var buffer bytes.Buffer
	if err := render(&buffer); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = buffer.WriteTo(w)
}
