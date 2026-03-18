package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"snippetbox.jgrecu.eu/internal/assert"
)

func TestPing(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/ping")

	assert.Equal(t, code, http.StatusOK)
	assert.Equal(t, body, "OK")
}

func TestSnippetView(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		wantCode int
		wantBody string
	}{
		{
			name:     "Valid ID",
			urlPath:  "/snippet/view/1",
			wantCode: http.StatusOK,
			wantBody: "An old silent pond...",
		},
		{
			name:     "Non-existent ID",
			urlPath:  "/snippet/view/2",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Negative ID",
			urlPath:  "/snippet/view/-1",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Decimal ID",
			urlPath:  "/snippet/view/1.23",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "String ID",
			urlPath:  "/snippet/view/foo",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Empty ID",
			urlPath:  "/snippet/view/",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, body := ts.get(t, tt.urlPath)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}
		})
	}
}

func TestUserSignup(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	const (
		validName     = "Bob"
		validPassword = "validPa$$word"
		validEmail    = "bob@example.com"
		formTag       = "<form action='/user/signup' method='POST' novalidate>"
	)

	tests := []struct {
		name         string
		userName     string
		userEmail    string
		userPassword string
		useValidCSRF bool
		wantCode     int
		wantFormTag  string
	}{
		{
			name:         "Valid submission",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusSeeOther,
		},
		{
			name:         "Invalid CSRF Token",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: validPassword,
			useValidCSRF: false,
			wantCode:     http.StatusBadRequest,
		},
		{
			name:         "Empty name",
			userName:     "",
			userEmail:    validEmail,
			userPassword: validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Empty email",
			userName:     validName,
			userEmail:    "",
			userPassword: validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Empty password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Invalid email",
			userName:     validName,
			userEmail:    "bob@example.",
			userPassword: validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Short password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "pa$$",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Duplicate email",
			userName:     validName,
			userEmail:    "dupe@example.com",
			userPassword: validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh CSRF token for each test
			_, _, body := ts.get(t, "/user/signup")
			csrfToken := extractCSRFToken(t, body)

			form := url.Values{}
			form.Add("name", tt.userName)
			form.Add("email", tt.userEmail)
			form.Add("password", tt.userPassword)
			if tt.useValidCSRF {
				form.Add("csrf_token", csrfToken)
			} else {
				form.Add("csrf_token", "wrongToken")
			}

			code, _, body := ts.postForm(t, "/user/signup", form)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantFormTag != "" {
				assert.StringContains(t, body, tt.wantFormTag)
			}
		})
	}
}

// TestHome tests the home page handler
func TestHome(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/")

	assert.Equal(t, code, http.StatusOK)
	assert.StringContains(t, body, "An old silent pond")
}

// TestSnippetViewEdgeCases tests additional edge cases for snippet view
func TestSnippetViewEdgeCases(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		wantCode int
	}{
		{
			name:     "Zero ID",
			urlPath:  "/snippet/view/0",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Very large ID",
			urlPath:  "/snippet/view/999999999",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "SQL injection attempt",
			urlPath:  "/snippet/view/1;DROP TABLE snippets",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "XSS attempt in ID",
			urlPath:  "/snippet/view/<script>alert('xss')</script>",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Whitespace ID",
			urlPath:  "/snippet/view/%20%20",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Null byte ID",
			urlPath:  "/snippet/view/1%00",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := ts.get(t, tt.urlPath)
			assert.Equal(t, code, tt.wantCode)
		})
	}
}

// TestUserLogin tests the login page and login functionality
func TestUserLogin(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	// Test GET request to login page
	t.Run("Login page renders", func(t *testing.T) {
		code, _, body := ts.get(t, "/user/login")
		assert.Equal(t, code, http.StatusOK)
		assert.StringContains(t, body, "<form action='/user/login' method='POST' novalidate>")
	})

	const (
		validEmail    = "alice@example.com"
		validPassword = "pa$$word"
		formTag       = "<form action='/user/login' method='POST' novalidate>"
	)

	tests := []struct {
		name         string
		userEmail    string
		password     string
		useValidCSRF bool
		wantCode     int
		wantBody     string
		wantLocation string
	}{
		{
			name:         "Valid credentials",
			userEmail:    validEmail,
			password:     validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusSeeOther,
			wantLocation: "/snippet/create",
		},
		{
			name:         "Invalid password",
			userEmail:    validEmail,
			password:     "wrongPassword",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     "Email or password is incorrect",
		},
		{
			name:         "Invalid email",
			userEmail:    "wrong@example.com",
			password:     validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     "Email or password is incorrect",
		},
		{
			name:         "Empty email",
			userEmail:    "",
			password:     validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     formTag,
		},
		{
			name:         "Empty password",
			userEmail:    validEmail,
			password:     "",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     formTag,
		},
		{
			name:         "Invalid email format",
			userEmail:    "not-an-email",
			password:     validPassword,
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     formTag,
		},
		{
			name:         "Invalid CSRF token",
			userEmail:    validEmail,
			password:     validPassword,
			useValidCSRF: false,
			wantCode:     http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh CSRF token for each test
			_, _, body := ts.get(t, "/user/login")
			csrfToken := extractCSRFToken(t, body)

			form := url.Values{}
			form.Add("email", tt.userEmail)
			form.Add("password", tt.password)
			if tt.useValidCSRF {
				form.Add("csrf_token", csrfToken)
			} else {
				form.Add("csrf_token", "invalidToken")
			}

			code, headers, body := ts.postForm(t, "/user/login", form)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}

			if tt.wantLocation != "" {
				assert.Equal(t, headers.Get("Location"), tt.wantLocation)
			}
		})
	}
}

// TestUserLogout tests the logout functionality
func TestUserLogout(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	// First, login to get an authenticated session
	_, _, body := ts.get(t, "/user/login")
	csrfToken := extractCSRFToken(t, body)

	form := url.Values{}
	form.Add("email", "alice@example.com")
	form.Add("password", "pa$$word")
	form.Add("csrf_token", csrfToken)
	ts.postForm(t, "/user/login", form)

	// Get a new CSRF token for the logout request
	_, _, body = ts.get(t, "/")
	csrfToken = extractCSRFToken(t, body)

	t.Run("Successful logout", func(t *testing.T) {
		form := url.Values{}
		form.Add("csrf_token", csrfToken)

		code, headers, _ := ts.postForm(t, "/user/logout", form)

		assert.Equal(t, code, http.StatusSeeOther)
		assert.Equal(t, headers.Get("Location"), "/")
	})

	t.Run("Logout without authentication redirects to login", func(t *testing.T) {
		// Create a fresh test server without logging in
		app := newTestApplication(t)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		_, _, body := ts.get(t, "/user/login")
		csrfToken := extractCSRFToken(t, body)

		form := url.Values{}
		form.Add("csrf_token", csrfToken)

		code, headers, _ := ts.postForm(t, "/user/logout", form)

		assert.Equal(t, code, http.StatusSeeOther)
		assert.Equal(t, headers.Get("Location"), "/user/login")
	})
}

// TestSnippetCreate tests the snippet creation page and functionality
func TestSnippetCreate(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	t.Run("Unauthenticated user redirected to login", func(t *testing.T) {
		code, headers, _ := ts.get(t, "/snippet/create")

		assert.Equal(t, code, http.StatusSeeOther)
		assert.Equal(t, headers.Get("Location"), "/user/login")
	})

	// Login first
	_, _, body := ts.get(t, "/user/login")
	csrfToken := extractCSRFToken(t, body)

	form := url.Values{}
	form.Add("email", "alice@example.com")
	form.Add("password", "pa$$word")
	form.Add("csrf_token", csrfToken)
	ts.postForm(t, "/user/login", form)

	t.Run("Authenticated user can access create page", func(t *testing.T) {
		code, _, body := ts.get(t, "/snippet/create")

		assert.Equal(t, code, http.StatusOK)
		assert.StringContains(t, body, "<form action='/snippet/create' method='POST'>")
	})

	const formTag = "<form action='/snippet/create' method='POST'>"

	tests := []struct {
		name         string
		title        string
		content      string
		expires      string
		useValidCSRF bool
		wantCode     int
		wantBody     string
	}{
		{
			name:         "Valid submission",
			title:        "Test Title",
			content:      "Test content for the snippet",
			expires:      "365",
			useValidCSRF: true,
			wantCode:     http.StatusSeeOther,
		},
		{
			name:         "Empty title",
			title:        "",
			content:      "Test content",
			expires:      "365",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     "This field cannot be blank",
		},
		{
			name:         "Empty content",
			title:        "Test Title",
			content:      "",
			expires:      "365",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     "This field cannot be blank",
		},
		{
			name:         "Title too long",
			title:        strings.Repeat("a", 101),
			content:      "Test content",
			expires:      "365",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     "This field cannot be more than 100 characters long",
		},
		{
			name:         "Invalid expiry value",
			title:        "Test Title",
			content:      "Test content",
			expires:      "30",
			useValidCSRF: true,
			wantCode:     http.StatusUnprocessableEntity,
			wantBody:     "This field must equal 1, 7 or 365",
		},
		{
			name:         "Expiry 1 day",
			title:        "Test Title",
			content:      "Test content",
			expires:      "1",
			useValidCSRF: true,
			wantCode:     http.StatusSeeOther,
		},
		{
			name:         "Expiry 7 days",
			title:        "Test Title",
			content:      "Test content",
			expires:      "7",
			useValidCSRF: true,
			wantCode:     http.StatusSeeOther,
		},
		{
			name:         "Invalid CSRF token",
			title:        "Test Title",
			content:      "Test content",
			expires:      "365",
			useValidCSRF: false,
			wantCode:     http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh CSRF token for each test
			_, _, body := ts.get(t, "/snippet/create")
			csrfToken := extractCSRFToken(t, body)

			form := url.Values{}
			form.Add("title", tt.title)
			form.Add("content", tt.content)
			form.Add("expires", tt.expires)
			if tt.useValidCSRF {
				form.Add("csrf_token", csrfToken)
			} else {
				form.Add("csrf_token", "invalid")
			}

			code, _, body := ts.postForm(t, "/snippet/create", form)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}
		})
	}
}

// TestProtectedRoutes tests that protected routes require authentication
func TestProtectedRoutes(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		method   string
		wantCode int
		wantLoc  string
	}{
		{
			name:     "GET /snippet/create requires auth",
			urlPath:  "/snippet/create",
			method:   "GET",
			wantCode: http.StatusSeeOther,
			wantLoc:  "/user/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, headers, _ := ts.get(t, tt.urlPath)

			assert.Equal(t, code, tt.wantCode)
			assert.Equal(t, headers.Get("Location"), tt.wantLoc)
		})
	}
}

// TestNavLiveClass tests that the navigation "live" class is applied correctly
func TestNavLiveClass(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	tests := []struct {
		name         string
		urlPath      string
		wantLiveLink string
	}{
		{
			name:         "Home page has live class on Home link",
			urlPath:      "/",
			wantLiveLink: `<a href='/' class='live'>Home</a>`,
		},
		{
			name:         "Signup page has live class on Signup link",
			urlPath:      "/user/signup",
			wantLiveLink: `<a href='/user/signup' class='live'>Signup</a>`,
		},
		{
			name:         "Login page has live class on Login link",
			urlPath:      "/user/login",
			wantLiveLink: `<a href='/user/login' class='live'>Login</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, body := ts.get(t, tt.urlPath)
			assert.StringContains(t, body, tt.wantLiveLink)
		})
	}

	// Test authenticated user navigation
	t.Run("Create snippet page has live class when authenticated", func(t *testing.T) {
		// Login first
		_, _, body := ts.get(t, "/user/login")
		csrfToken := extractCSRFToken(t, body)

		form := url.Values{}
		form.Add("email", "alice@example.com")
		form.Add("password", "pa$$word")
		form.Add("csrf_token", csrfToken)
		ts.postForm(t, "/user/login", form)

		_, _, body = ts.get(t, "/snippet/create")
		assert.StringContains(t, body, `<a href='/snippet/create' class='live'>Create snippet</a>`)
	})
}

// TestFlashMessages tests that flash messages are displayed correctly
func TestFlashMessages(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	t.Run("Flash message after successful signup", func(t *testing.T) {
		_, _, body := ts.get(t, "/user/signup")
		csrfToken := extractCSRFToken(t, body)

		form := url.Values{}
		form.Add("name", "Test User")
		form.Add("email", "test@example.com")
		form.Add("password", "password123")
		form.Add("csrf_token", csrfToken)

		// Submit signup form - should redirect to login
		code, headers, _ := ts.postForm(t, "/user/signup", form)
		assert.Equal(t, code, http.StatusSeeOther)
		assert.Equal(t, headers.Get("Location"), "/user/login")

		// Follow redirect to login page - should show flash message
		_, _, body = ts.get(t, "/user/login")
		assert.StringContains(t, body, "Your signup was successful. Please log in.")
	})

	t.Run("Flash message after logout", func(t *testing.T) {
		// Login first
		_, _, body := ts.get(t, "/user/login")
		csrfToken := extractCSRFToken(t, body)

		form := url.Values{}
		form.Add("email", "alice@example.com")
		form.Add("password", "pa$$word")
		form.Add("csrf_token", csrfToken)
		ts.postForm(t, "/user/login", form)

		// Get new CSRF token and logout
		_, _, body = ts.get(t, "/")
		csrfToken = extractCSRFToken(t, body)

		form = url.Values{}
		form.Add("csrf_token", csrfToken)
		ts.postForm(t, "/user/logout", form)

		// Check homepage for flash message
		_, _, body = ts.get(t, "/")
		assert.StringContains(t, body, "You&#39;ve been logged out successfully!")
	})
}

// TestSecurityHeaders tests that security headers are present in responses
func TestSecurityHeaders(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	_, headers, _ := ts.get(t, "/")

	tests := []struct {
		header string
		want   string
	}{
		{
			header: "Content-Security-Policy",
			want:   "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com",
		},
		{
			header: "Referrer-Policy",
			want:   "origin-when-cross-origin",
		},
		{
			header: "X-Content-Type-Options",
			want:   "nosniff",
		},
		{
			header: "X-Frame-Options",
			want:   "deny",
		},
		{
			header: "X-XSS-Protection",
			want:   "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			assert.Equal(t, headers.Get(tt.header), tt.want)
		})
	}
}

// TestCacheControlOnProtectedRoutes tests Cache-Control header on protected routes
func TestCacheControlOnProtectedRoutes(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	// Login first
	_, _, body := ts.get(t, "/user/login")
	csrfToken := extractCSRFToken(t, body)

	form := url.Values{}
	form.Add("email", "alice@example.com")
	form.Add("password", "pa$$word")
	form.Add("csrf_token", csrfToken)
	ts.postForm(t, "/user/login", form)

	// Access protected route
	_, headers, _ := ts.get(t, "/snippet/create")

	assert.Equal(t, headers.Get("Cache-Control"), "no-store")
}
