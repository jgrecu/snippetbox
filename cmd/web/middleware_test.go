package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"snippetbox.jgrecu.eu/internal/assert"
)

func TestCommonHeaders(t *testing.T) {
	// Initialize a new httptest.ResponseRecorder and dummy http.Request.
	rr := httptest.NewRecorder()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock HTTP handler that we can pass to our commonHeaders
	// middleware, which writes a 200 status code and an "OK" response body.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Pass the mock HTTP handler to our commonHeaders middleware. Because
	// commonHeaders *returns* a http.Handler we can call its ServeHTTP()
	// method, passing in the http.ResponseRecorder and dummy http.Request to
	// execute it.
	commonHeaders(next).ServeHTTP(rr, r)

	// Call the Result() method on the http.ResponseRecorder to get the results
	// of the test.
	rs := rr.Result()

	// Check that the middleware has correctly set the Content-Security-Policy
	// header on the response.
	expectedValue := "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com"
	assert.Equal(t, rs.Header.Get("Content-Security-Policy"), expectedValue)

	// Check that the middleware has correctly set the Referrer-Policy
	// header on the response.
	expectedValue = "origin-when-cross-origin"
	assert.Equal(t, rs.Header.Get("Referrer-Policy"), expectedValue)

	// Check that the middleware has correctly set the X-Content-Type-Options
	// header on the response.
	expectedValue = "nosniff"
	assert.Equal(t, rs.Header.Get("X-Content-Type-Options"), expectedValue)

	// Check that the middleware has correctly set the X-Frame-Options header
	// on the response.
	expectedValue = "deny"
	assert.Equal(t, rs.Header.Get("X-Frame-Options"), expectedValue)

	// Check that the middleware has correctly set the X-XSS-Protection header
	// on the response
	expectedValue = "0"
	assert.Equal(t, rs.Header.Get("X-XSS-Protection"), expectedValue)

	// Check that the middleware has correctly set the Server header on the
	// response.
	expectedValue = "Go"
	assert.Equal(t, rs.Header.Get("Server"), expectedValue)

	// Check that the middleware has correctly called the next handler in line
	// and the response status code and body are as expected.
	assert.Equal(t, rs.StatusCode, http.StatusOK)

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.TrimSpace(body)

	assert.Equal(t, string(body), "OK")
}

// TestRecoverPanic tests that the recoverPanic middleware catches panics
func TestRecoverPanic(t *testing.T) {
	app := newTestApplication(t)

	t.Run("Recovers from panic", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Create a handler that panics
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		app.recoverPanic(next).ServeHTTP(rr, r)

		rs := rr.Result()

		assert.Equal(t, rs.StatusCode, http.StatusInternalServerError)
		assert.Equal(t, rs.Header.Get("Connection"), "close")
	})

	t.Run("Normal request passes through", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		app.recoverPanic(next).ServeHTTP(rr, r)

		rs := rr.Result()

		assert.Equal(t, rs.StatusCode, http.StatusOK)
	})
}

// TestRequireAuthentication tests the requireAuthentication middleware
func TestRequireAuthentication(t *testing.T) {
	app := newTestApplication(t)

	t.Run("Unauthenticated user redirected", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/snippet/create", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Add session context
		ctx := app.sessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("OK"))
			})
			app.requireAuthentication(next).ServeHTTP(w, r)
		}))

		ctx.ServeHTTP(rr, r)

		rs := rr.Result()

		assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
		assert.Equal(t, rs.Header.Get("Location"), "/user/login")
	})

	t.Run("Authenticated user passes through", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/snippet/create", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Set authenticated context
		ctx := context.WithValue(r.Context(), isAuthenticatedContextKey, true)
		r = r.WithContext(ctx)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		app.requireAuthentication(next).ServeHTTP(rr, r)

		rs := rr.Result()

		assert.Equal(t, rs.StatusCode, http.StatusOK)
		assert.Equal(t, rs.Header.Get("Cache-Control"), "no-store")
	})
}

// TestAuthenticate tests the authenticate middleware
func TestAuthenticate(t *testing.T) {
	app := newTestApplication(t)

	t.Run("No session ID passes through without authentication", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		var isAuth bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAuth = app.isAuthenticated(r)
			w.Write([]byte("OK"))
		})

		// Wrap with session manager
		handler := app.sessionManager.LoadAndSave(app.authenticate(next))
		handler.ServeHTTP(rr, r)

		assert.Equal(t, isAuth, false)
	})
}

// TestLogRequest tests the logRequest middleware
func TestLogRequest(t *testing.T) {
	app := newTestApplication(t)

	rr := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test-path", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.RemoteAddr = "127.0.0.1:1234"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	app.logRequest(next).ServeHTTP(rr, r)

	rs := rr.Result()

	// Verify the request passed through correctly
	assert.Equal(t, rs.StatusCode, http.StatusOK)

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(bytes.TrimSpace(body)), "OK")
}
