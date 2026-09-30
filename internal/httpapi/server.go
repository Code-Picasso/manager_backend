package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"manager-backend/internal/auth"
	"manager-backend/internal/models"
	"manager-backend/internal/store"
)

// Server holds the dependencies every handler needs.
type Server struct {
	store *store.Store
}

// New returns a Server backed by s.
func New(s *store.Store) *Server {
	return &Server{store: s}
}

type contextKey int

const currentUserKey contextKey = iota

// Routes builds the router with all API endpoints and middleware.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/register", s.register)
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/reset-password", s.resetPassword)

	mux.Handle("GET /api/user", s.requireAuth(http.HandlerFunc(s.user)))
	mux.Handle("PUT /api/user", s.requireAuth(http.HandlerFunc(s.updateProfile)))
	mux.Handle("DELETE /api/user", s.requireAuth(http.HandlerFunc(s.deactivate)))
	mux.Handle("POST /api/logout", s.requireAuth(http.HandlerFunc(s.logout)))
	mux.Handle("POST /api/change-password", s.requireAuth(http.HandlerFunc(s.changePassword)))

	mux.Handle("GET /api/tasks", s.requireAuth(http.HandlerFunc(s.listTasks)))
	mux.Handle("POST /api/tasks", s.requireAuth(http.HandlerFunc(s.createTask)))
	mux.Handle("GET /api/tasks/{task}", s.requireAuth(http.HandlerFunc(s.showTask)))
	mux.Handle("PUT /api/tasks/{task}", s.requireAuth(http.HandlerFunc(s.updateTask)))
	mux.Handle("DELETE /api/tasks/{task}", s.requireAuth(http.HandlerFunc(s.deleteTask)))
	mux.Handle("POST /api/tasks/{task}/restore", s.requireAuth(http.HandlerFunc(s.restoreTask)))
	mux.Handle("POST /api/tasks/{task}/subtasks/{subTask}/toggle", s.requireAuth(http.HandlerFunc(s.toggleSubTask)))

	mux.Handle("GET /api/notes", s.requireAuth(http.HandlerFunc(s.listNotes)))
	mux.Handle("POST /api/notes", s.requireAuth(http.HandlerFunc(s.createNote)))
	mux.Handle("GET /api/notes/{note}", s.requireAuth(http.HandlerFunc(s.showNote)))
	mux.Handle("PUT /api/notes/{note}", s.requireAuth(http.HandlerFunc(s.updateNote)))
	mux.Handle("DELETE /api/notes/{note}", s.requireAuth(http.HandlerFunc(s.deleteNote)))

	mux.Handle("GET /api/alerts", s.requireAuth(http.HandlerFunc(s.listAlerts)))
	mux.Handle("POST /api/alerts/read-all", s.requireAuth(http.HandlerFunc(s.markAllRead)))

	return s.rateLimit(mux)
}

// requireAuth resolves the bearer token to a user or answers 401.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		u, err := s.store.UserForToken(r.Context(), auth.HashToken(token))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Unauthenticated.")
			return
		}
		ctx := context.WithValue(r.Context(), currentUserKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUser(r *http.Request) models.User {
	return r.Context().Value(currentUserKey).(models.User)
}

// bearerToken extracts the token from the Authorization header.
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return h[len(prefix):], true
	}
	return "", false
}

// issueToken creates and stores a new bearer token for a user.
func (s *Server) issueToken(ctx context.Context, userID int64) (string, error) {
	plain, hash, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateToken(ctx, userID, hash); err != nil {
		return "", err
	}
	return plain, nil
}

// syncAlerts reconciles alerts, treating failure as non-fatal.
func (s *Server) syncAlerts(ctx context.Context, userID int64) {
	if err := s.store.SyncAlerts(ctx, userID, time.Now().UTC()); err != nil {
		log.Printf("alert sync failed: %v", err)
	}
}

// readBody decodes the JSON request body, returning an empty map when absent.
func readBody(r *http.Request) map[string]any {
	var m map[string]any
	_ = json.NewDecoder(r.Body).Decode(&m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"message": ""})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "Server Error.")
}

func writeValidationError(w http.ResponseWriter, errors validationErrors) {
	writeJSON(w, http.StatusUnprocessableEntity, validationBody(errors))
}

func validationBody(errors validationErrors) map[string]any {
	all := make([]string, 0)
	for _, msgs := range errors {
		all = append(all, msgs...)
	}

	message := "The given data was invalid."
	if len(all) > 0 {
		message = all[0]
		if rest := len(all) - 1; rest == 1 {
			message += " (and 1 more error)"
		} else if rest > 1 {
			message += fmt.Sprintf(" (and %d more errors)", rest)
		}
	}
	return map[string]any{"message": message, "errors": errors}
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]*rateWindow
}

type rateWindow struct {
	start time.Time
	count int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{windows: make(map[string]*rateWindow)}
}

func (rl *rateLimiter) allow(key string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	w, ok := rl.windows[key]
	if !ok || now.Sub(w.start) >= time.Minute {
		rl.windows[key] = &rateWindow{start: now, count: 1}
		return true
	}
	if w.count >= 60 {
		return false
	}
	w.count++
	return true
}

// rateLimit caps each client at 60 requests per minute.
func (s *Server) rateLimit(next http.Handler) http.Handler {
	rl := newRateLimiter()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r), time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "Too Many Attempts.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isNotFound reports whether err is the store's not-found sentinel.
func isNotFound(err error) bool {
	return errors.Is(err, store.ErrNotFound)
}
