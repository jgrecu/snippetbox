# Snippetbox

A secure, full-featured web application for sharing text snippets, built with Go. Inspired by services like Pastebin and GitHub Gists.

## Features

- **User Authentication** - Secure signup, login, and logout with bcrypt password hashing
- **Snippet Management** - Create, view, and share text snippets with configurable expiration (1 day, 7 days, or 1 year)
- **Session Management** - Server-side sessions stored in MySQL with automatic expiry
- **Security Hardened** - CSRF protection, secure headers, TLS encryption, and SQL injection prevention
- **Clean Architecture** - Follows Go best practices with separation of concerns

## Tech Stack

- **Go 1.26** - Backend language
- **MySQL** - Database for snippets, users, and sessions
- **HTML Templates** - Server-side rendering with Go's `html/template`
- **TLS/HTTPS** - Encrypted connections with modern cipher suites

### Dependencies

| Package | Purpose |
|---------|---------|
| [alexedwards/scs](https://github.com/alexedwards/scs) | Session management |
| [justinas/alice](https://github.com/justinas/alice) | Middleware chaining |
| [justinas/nosurf](https://github.com/justinas/nosurf) | CSRF protection |
| [go-playground/form](https://github.com/go-playground/form) | Form decoding |
| [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | MySQL driver |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | Password hashing (bcrypt) |

## Project Structure

```
snippetbox/
├── cmd/
│   └── web/              # Application entry point and web handlers
│       ├── main.go       # Server setup and configuration
│       ├── handlers.go   # HTTP request handlers
│       ├── routes.go     # URL routing
│       ├── middleware.go # HTTP middleware (auth, logging, recovery)
│       ├── helpers.go    # Helper functions
│       └── templates.go  # Template caching and rendering
├── internal/
│   ├── models/           # Database models
│   │   ├── snippets.go   # Snippet CRUD operations
│   │   ├── users.go      # User authentication
│   │   └── mocks/        # Mock models for testing
│   ├── validator/        # Form validation
│   └── assert/           # Test assertion helpers
├── ui/
│   ├── html/             # Go templates
│   │   ├── base.tmpl     # Base layout
│   │   ├── pages/        # Page templates
│   │   └── partials/     # Reusable template partials
│   └── static/           # Static assets (CSS, images)
└── tls/                  # TLS certificates (not in repo)
```

## Getting Started

### Prerequisites

- Go 1.26 or later
- MySQL 8.0 or later

### Database Setup

1. Create the database and user:

```sql
CREATE DATABASE snippetbox CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'web'@'localhost' IDENTIFIED BY 'pass';
GRANT SELECT, INSERT, UPDATE, DELETE ON snippetbox.* TO 'web'@'localhost';
```

2. Create the required tables:

```sql
USE snippetbox;

CREATE TABLE snippets (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created DATETIME NOT NULL,
    expires DATETIME NOT NULL
);

CREATE INDEX idx_snippets_created ON snippets(created);

CREATE TABLE users (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    hashed_password CHAR(60) NOT NULL,
    created DATETIME NOT NULL
);

ALTER TABLE users ADD CONSTRAINT users_uc_email UNIQUE (email);

CREATE TABLE sessions (
    token CHAR(43) PRIMARY KEY,
    data BLOB NOT NULL,
    expiry TIMESTAMP(6) NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions(expiry);
```

### TLS Certificates

Generate self-signed certificates for local development:

```bash
mkdir -p tls
cd tls
go run /usr/local/go/src/crypto/tls/generate_cert.go --rsa-bits=2048 --host=localhost
```

### Running the Application

```bash
# Clone the repository
git clone https://github.com/jgrecu/snippetbox.git
cd snippetbox

# Install dependencies
go mod download

# Run the application
go run ./cmd/web

# Or with custom flags
go run ./cmd/web -addr=":8080" -dsn="user:password@/dbname?parseTime=true"
```

The application will be available at `https://localhost:4000`

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:4000` | HTTP network address |
| `-dsn` | `web:pass@/snippetbox?parseTime=true` | MySQL data source name |

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./cmd/web
go tool cover -html=coverage.out
```

Current test coverage: **73%**

## Security Features

- **HTTPS Only** - All traffic encrypted with TLS 1.2+
- **Secure Headers** - CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- **CSRF Protection** - SameSite cookies and token-based protection
- **Password Security** - bcrypt hashing with cost factor 12
- **SQL Injection Prevention** - Parameterized queries throughout
- **Session Security** - Secure, HttpOnly cookies with server-side storage
- **Input Validation** - Server-side validation for all user inputs

## License

This project is for educational purposes, based on the book "Let's Go" by Alex Edwards.
