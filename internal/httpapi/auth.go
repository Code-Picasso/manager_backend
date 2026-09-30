package httpapi

import (
	"net/http"
	"strings"

	"manager-backend/internal/auth"
)

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	m := readBody(r)
	e := validationErrors{}

	name, nameOK := requiredString(m, "name", e)
	if nameOK {
		maxLength(name, "name", 255, e)
	}

	email, emailOK := requiredEmail(m, "email", e)
	if emailOK {
		email = strings.ToLower(strings.TrimSpace(email))
		maxLength(email, "email", 255, e)
		if len(e["email"]) == 0 {
			_, err := s.store.FindUserByEmail(r.Context(), email)
			if err != nil && !isNotFound(err) {
				serverError(w, err)
				return
			}
			if err == nil {
				e.add("email", "The email has already been taken.")
			}
		}
	}

	password, passwordOK := requiredString(m, "password", e)
	if passwordOK {
		if msg := auth.StrongPasswordError(password); msg != "" {
			e.add("password", msg)
		}
	}

	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		serverError(w, err)
		return
	}

	user, err := s.store.CreateUser(r.Context(), name, email, hash)
	if err != nil {
		serverError(w, err)
		return
	}

	token, err := s.issueToken(r.Context(), user.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":  newUserResponse(user),
		"token": token,
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	m := readBody(r)
	e := validationErrors{}

	email, emailOK := requiredEmail(m, "email", e)
	if emailOK {
		email = strings.ToLower(strings.TrimSpace(email))
	}
	password, _ := requiredString(m, "password", e)

	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	user, err := s.store.FindUserByEmail(r.Context(), email)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusUnauthorized, "The provided credentials are incorrect.")
			return
		}
		serverError(w, err)
		return
	}
	if !auth.CheckPassword(user.Password, password) {
		writeError(w, http.StatusUnauthorized, "The provided credentials are incorrect.")
		return
	}

	token, err := s.issueToken(r.Context(), user.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":  newUserResponse(user),
		"token": token,
	})
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	m := readBody(r)
	e := validationErrors{}

	email, emailOK := requiredEmail(m, "email", e)
	if emailOK {
		email = strings.ToLower(strings.TrimSpace(email))
	}

	password, passwordOK := requiredString(m, "password", e)
	if passwordOK {
		if msg := auth.StrongPasswordError(password); msg != "" {
			e.add("password", msg)
		}
	}

	var userID int64
	if len(e["email"]) == 0 {
		user, err := s.store.FindUserByEmail(r.Context(), email)
		if err != nil {
			if isNotFound(err) {
				e.add("email", "The selected email is invalid.")
			} else {
				serverError(w, err)
				return
			}
		} else {
			userID = user.ID
		}
	}

	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := s.store.UpdateUserPassword(r.Context(), userID, hash); err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password reset."})
}

func (s *Server) user(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, newUserResponse(currentUser(r)))
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	m := readBody(r)
	e := validationErrors{}

	name, nameOK := requiredString(m, "name", e)
	if nameOK {
		maxLength(name, "name", 255, e)
	}
	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	updated, err := s.store.UpdateUserName(r.Context(), u.ID, name)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newUserResponse(updated))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	token, _ := bearerToken(r)
	if err := s.store.DeleteToken(r.Context(), auth.HashToken(token)); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out."})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	m := readBody(r)
	e := validationErrors{}

	currentPassword, _ := requiredString(m, "current_password", e)

	password, passwordOK := requiredString(m, "password", e)
	if passwordOK {
		if msg := auth.StrongPasswordError(password); msg != "" {
			e.add("password", msg)
		}
	}

	if len(e) == 0 && !auth.CheckPassword(u.Password, currentPassword) {
		e.add("current_password", "The current password is incorrect.")
	}

	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := s.store.UpdateUserPassword(r.Context(), u.ID, hash); err != nil {
		serverError(w, err)
		return
	}

	token, _ := bearerToken(r)
	if err := s.store.DeleteOtherTokens(r.Context(), u.ID, auth.HashToken(token)); err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password changed."})
}

func (s *Server) deactivate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if err := s.store.DeleteUser(r.Context(), u.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Account deactivated."})
}
