package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"manager-backend/internal/auth"
	"manager-backend/internal/database"
	"manager-backend/internal/httpapi"
	"manager-backend/internal/models"
	"manager-backend/internal/store"
)

var (
	testPool *pgxpool.Pool
	nameSeq  atomic.Int64
)

func TestMain(m *testing.M) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		raw = "postgres://manager:secret@localhost:5433/manager_test?sslmode=disable"
	}

	adminURL, dbName, err := splitDBName(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse TEST_DATABASE_URL: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	adminPool, err := database.Open(ctx, adminURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to admin db: %v\n", err)
		os.Exit(1)
	}
	if _, err := adminPool.Exec(ctx, "DROP DATABASE IF EXISTS "+dbName+" WITH (FORCE)"); err != nil {
		fmt.Fprintf(os.Stderr, "drop test db: %v\n", err)
		os.Exit(1)
	}
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		fmt.Fprintf(os.Stderr, "create test db: %v\n", err)
		os.Exit(1)
	}
	adminPool.Close()

	pool, err := database.Open(ctx, raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to test db: %v\n", err)
		os.Exit(1)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "migrate test db: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func splitDBName(raw string) (adminURL, dbName string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	dbName = strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		dbName = "postgres"
	}
	u.Path = "/postgres"
	return u.String(), dbName, nil
}

func resetDB() {
	if _, err := testPool.Exec(context.Background(),
		`TRUNCATE users, personal_access_tokens, tasks, sub_tasks, notes, alerts RESTART IDENTITY CASCADE`); err != nil {
		panic(err)
	}
}

func uniqueName() string {
	return fmt.Sprintf("user%d", nameSeq.Add(1))
}

func todayStr() string {
	return time.Now().UTC().Format("2006-01-02")
}

func yesterdayStr() string {
	return time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
}

type testEnv struct {
	t      *testing.T
	server *httptest.Server
	store  *store.Store
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	resetDB()
	s := store.New(testPool)
	server := httptest.NewServer(httpapi.New(s).Routes())
	t.Cleanup(server.Close)
	return &testEnv{t: t, server: server, store: s}
}

// do issues a request and returns the status code and raw body.
func (e *testEnv) do(method, path, token string, body any) (int, []byte) {
	e.t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		e.t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := e.server.Client().Do(req)
	if err != nil {
		e.t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, data
}

// createUser creates a user with password "password" and returns it with a token.
func (e *testEnv) createUser() (models.User, string) {
	e.t.Helper()
	hash, err := auth.HashPassword("password")
	if err != nil {
		e.t.Fatalf("hash password: %v", err)
	}
	u, err := e.store.CreateUser(context.Background(), uniqueName(), hash)
	if err != nil {
		e.t.Fatalf("create user: %v", err)
	}

	plain, tokenHash, err := auth.NewToken()
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.store.CreateToken(context.Background(), u.ID, tokenHash); err != nil {
		e.t.Fatal(err)
	}
	return u, plain
}

// registerUser registers over HTTP and returns the token.
func (e *testEnv) registerUser(name, password string) string {
	e.t.Helper()
	status, body := e.do("POST", "/api/register", "", map[string]any{
		"name": name, "password": password,
	})
	if status != http.StatusCreated {
		e.t.Fatalf("register status = %d, body = %s", status, body)
	}
	return jsonPath(e.t, body, "token").(string)
}

// storeTask creates a task directly in the database.
func (e *testEnv) storeTask(userID int64, task models.Task) models.Task {
	e.t.Helper()
	task.UserID = userID
	if task.Title == "" {
		task.Title = "Default task"
	}
	if task.Category == "" {
		task.Category = "work"
	}
	if task.Status == "" {
		task.Status = "today"
	}
	if task.Date == "" {
		task.Date = todayStr()
	}
	if task.StartTime == "" {
		task.StartTime = "09:00"
	}
	if task.EndTime == "" {
		task.EndTime = "10:00"
	}
	created, err := e.store.CreateTask(context.Background(), task)
	if err != nil {
		e.t.Fatalf("create task: %v", err)
	}
	return created
}

func validTaskPayload(overrides map[string]any) map[string]any {
	payload := map[string]any{
		"title":       "Write tests",
		"description": "Cover the API",
		"category":    "work",
		"status":      "today",
		"date":        "2026-09-15",
		"start_time":  "09:00",
		"end_time":    "10:00",
		"sub_tasks": []any{
			map[string]any{"title": "First", "is_done": false},
			map[string]any{"title": "Second", "is_done": false},
		},
	}
	for k, v := range overrides {
		payload[k] = v
	}
	return payload
}

// jsonPath decodes body and walks the dotted path of map keys and array indices.
func jsonPath(t *testing.T, body []byte, path ...string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("invalid JSON %q: %v", body, err)
	}
	return lookup(t, v, path)
}

func lookup(t *testing.T, v any, path []string) any {
	t.Helper()
	cur := v
	for _, p := range path {
		switch node := cur.(type) {
		case map[string]any:
			var ok bool
			cur, ok = node[p]
			if !ok {
				t.Fatalf("path %v: key %q missing in %v", path, p, node)
			}
		case []any:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(node) {
				t.Fatalf("path %v: bad index %q in %v", path, p, node)
			}
			cur = node[idx]
		default:
			t.Fatalf("path %v: cannot descend into %T", path, cur)
		}
	}
	return cur
}

func jsonCount(t *testing.T, body []byte) int {
	t.Helper()
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("invalid JSON %q: %v", body, err)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("expected JSON array, got %T", v)
	}
	return len(arr)
}

// jsonBytes re-encodes the value at a path so nested helpers can consume it.
func jsonBytes(t *testing.T, body []byte, path ...string) []byte {
	t.Helper()
	b, err := json.Marshal(jsonPath(t, body, path...))
	if err != nil {
		t.Fatalf("marshal path %v: %v", path, err)
	}
	return b
}

func jsonErrors(t *testing.T, body []byte) map[string][]string {
	t.Helper()
	v := jsonPath(t, body, "errors")
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("errors is not a map: %T", v)
	}
	out := make(map[string][]string)
	for k, val := range m {
		if arr, ok := val.([]any); ok {
			for _, msg := range arr {
				if s, ok := msg.(string); ok {
					out[k] = append(out[k], s)
				}
			}
		}
	}
	return out
}
