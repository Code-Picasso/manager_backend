package httpapi_test

import (
	"net/http"
	"testing"

	"manager-backend/internal/auth"
)

func TestRegister(t *testing.T) {
	env := newTestEnv(t)

	status, body := env.do("POST", "/api/register", "", map[string]any{
		"name": "Ada Lovelace", "email": "ada@example.com", "password": "Secret123!",
	})
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "user", "name") != "Ada Lovelace" {
		t.Fatalf("wrong name: %s", body)
	}
	if jsonPath(t, body, "token") == "" {
		t.Fatalf("missing token: %s", body)
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	env := newTestEnv(t)

	status, body := env.do("POST", "/api/register", "", map[string]any{
		"name": "Ada", "email": "ada@example.com", "password": "weak",
	})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["password"]; !ok {
		t.Fatalf("missing password error: %s", body)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser("Ada", "ada@example.com", "Secret123!")

	status, body := env.do("POST", "/api/register", "", map[string]any{
		"name": "Ada", "email": "ada@example.com", "password": "Secret123!",
	})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["email"]; !ok {
		t.Fatalf("missing email error: %s", body)
	}
}

func TestEmailIsNormalisedToLowercase(t *testing.T) {
	env := newTestEnv(t)

	env.registerUser("Ada", "ADA@Example.COM", "Secret123!")

	_, err := env.store.FindUserByEmail(t.Context(), "ada@example.com")
	if err != nil {
		t.Fatalf("lowercased email not found: %v", err)
	}
}

func TestLogin(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser("Ada", "ada@example.com", "Secret123!")

	status, body := env.do("POST", "/api/login", "", map[string]any{
		"email": "ada@example.com", "password": "Secret123!",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "user", "email") != "ada@example.com" {
		t.Fatalf("wrong email: %s", body)
	}
	if jsonPath(t, body, "token") == "" {
		t.Fatalf("missing token: %s", body)
	}
}

func TestLoginWithWrongPasswordFails(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser("Ada", "ada@example.com", "Secret123!")

	status, _ := env.do("POST", "/api/login", "", map[string]any{
		"email": "ada@example.com", "password": "WrongPass1!",
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestGuestsAreRejected(t *testing.T) {
	env := newTestEnv(t)

	if status, _ := env.do("GET", "/api/user", "", nil); status != http.StatusUnauthorized {
		t.Fatalf("GET /api/user status = %d, want 401", status)
	}
	if status, _ := env.do("GET", "/api/tasks", "", nil); status != http.StatusUnauthorized {
		t.Fatalf("GET /api/tasks status = %d, want 401", status)
	}
}

func TestAuthenticatedUserCanFetchProfile(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()

	status, body := env.do("GET", "/api/user", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "email") != u.Email {
		t.Fatalf("wrong email: %s", body)
	}
}

func TestUpdateProfile(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("PUT", "/api/user", token, map[string]any{"name": "New Name"})
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "name") != "New Name" {
		t.Fatalf("wrong name: %s", body)
	}
}

func TestLogoutTokenStopsWorking(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	if status, _ := env.do("POST", "/api/logout", token, nil); status != http.StatusOK {
		t.Fatalf("logout status = %d", status)
	}

	if status, _ := env.do("GET", "/api/user", token, nil); status != http.StatusUnauthorized {
		t.Fatalf("revoked token still works, status = %d", status)
	}
}

func TestResetPassword(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser("Ada", "ada@example.com", "Secret123!")

	status, _ := env.do("POST", "/api/reset-password", "", map[string]any{
		"email": "ada@example.com", "password": "NewSecret123!",
	})
	if status != http.StatusOK {
		t.Fatalf("reset status = %d", status)
	}

	status, _ = env.do("POST", "/api/login", "", map[string]any{
		"email": "ada@example.com", "password": "NewSecret123!",
	})
	if status != http.StatusOK {
		t.Fatalf("login with new password status = %d", status)
	}
}

func TestDeactivateAccount(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()

	if status, _ := env.do("DELETE", "/api/user", token, nil); status != http.StatusOK {
		t.Fatalf("deactivate status = %d", status)
	}

	if _, err := env.store.FindUserByID(t.Context(), u.ID); err == nil {
		t.Fatalf("user still exists after deactivate")
	}
}

func TestChangePassword(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()

	status, _ := env.do("POST", "/api/change-password", token, map[string]any{
		"current_password": "password", "password": "NewSecret123!",
	})
	if status != http.StatusOK {
		t.Fatalf("change-password status = %d", status)
	}

	status, _ = env.do("POST", "/api/login", "", map[string]any{
		"email": u.Email, "password": "NewSecret123!",
	})
	if status != http.StatusOK {
		t.Fatalf("login with new password status = %d", status)
	}
}

func TestChangePasswordRequiresCurrent(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/change-password", token, map[string]any{
		"current_password": "not-my-password", "password": "NewSecret123!",
	})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["current_password"]; !ok {
		t.Fatalf("missing current_password error: %s", body)
	}
}

func TestChangePasswordEnforcesStrength(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/change-password", token, map[string]any{
		"current_password": "password", "password": "weak",
	})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["password"]; !ok {
		t.Fatalf("missing password error: %s", body)
	}
}

func TestChangePasswordRevokesOtherSessionsOnly(t *testing.T) {
	env := newTestEnv(t)
	u, current := env.createUser()

	// Issue a second token for the same user.
	otherPlain, otherHash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := env.store.CreateToken(t.Context(), u.ID, otherHash); err != nil {
		t.Fatal(err)
	}

	status, _ := env.do("POST", "/api/change-password", current, map[string]any{
		"current_password": "password", "password": "NewSecret123!",
	})
	if status != http.StatusOK {
		t.Fatalf("change-password status = %d", status)
	}

	// The current session stays signed in.
	if status, _ := env.do("GET", "/api/user", current, nil); status != http.StatusOK {
		t.Fatalf("current token revoked, status = %d", status)
	}
	// The other token is revoked.
	if status, _ := env.do("GET", "/api/user", otherPlain, nil); status != http.StatusUnauthorized {
		t.Fatalf("other token still works, status = %d", status)
	}
}

func TestGuestsCannotChangePassword(t *testing.T) {
	env := newTestEnv(t)

	status, _ := env.do("POST", "/api/change-password", "", map[string]any{
		"current_password": "password", "password": "NewSecret123!",
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}
