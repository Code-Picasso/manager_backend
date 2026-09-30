package httpapi_test

import (
	"net/http"
	"testing"
)

func TestCreateNote(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/notes", token, map[string]any{
		"title": "Groceries", "content": "Milk, eggs",
	})
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "title") != "Groceries" {
		t.Fatalf("wrong title: %s", body)
	}
}

func TestListOwnNotes(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()

	for i := 0; i < 2; i++ {
		if _, err := env.store.CreateNote(t.Context(), u.ID, "Note", "Body"); err != nil {
			t.Fatal(err)
		}
	}

	status, body := env.do("GET", "/api/notes", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 2 {
		t.Fatalf("wrong count: %s", body)
	}
}

func TestNoteWithoutBody(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/notes", token, map[string]any{"title": "Ideas"})
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "content") != "" {
		t.Fatalf("content not empty: %s", body)
	}
}

func TestNoteBodyCanBeCleared(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	note, err := env.store.CreateNote(t.Context(), u.ID, "Note", "Something")
	if err != nil {
		t.Fatal(err)
	}

	status, body := env.do("PUT", "/api/notes/"+note.ID, token, map[string]any{"content": ""})
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "content") != "" {
		t.Fatalf("content not cleared: %s", body)
	}
}

func TestUpdateNote(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	note, err := env.store.CreateNote(t.Context(), u.ID, "Note", "Body")
	if err != nil {
		t.Fatal(err)
	}

	status, body := env.do("PUT", "/api/notes/"+note.ID, token, map[string]any{"content": "Updated body"})
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "content") != "Updated body" {
		t.Fatalf("wrong content: %s", body)
	}
}

func TestDeleteNote(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	note, err := env.store.CreateNote(t.Context(), u.ID, "Note", "Body")
	if err != nil {
		t.Fatal(err)
	}

	if status, _ := env.do("DELETE", "/api/notes/"+note.ID, token, nil); status != http.StatusNoContent {
		t.Fatalf("delete status = %d", status)
	}
	if status, _ := env.do("GET", "/api/notes/"+note.ID, token, nil); status != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", status)
	}
}

func TestCannotSeeAnotherUsersNote(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()
	other, _ := env.createUser()
	note, err := env.store.CreateNote(t.Context(), other.ID, "Other note", "Body")
	if err != nil {
		t.Fatal(err)
	}

	status, _ := env.do("GET", "/api/notes/"+note.ID, token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}
