package httpapi

import (
	"net/http"
	"unicode/utf8"
)

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	notes, err := s.store.ListNotes(r.Context(), u.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	resp := make([]noteResponse, 0, len(notes))
	for _, n := range notes {
		resp = append(resp, newNoteResponse(n))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	m := readBody(r)
	e := validationErrors{}

	title, titleOK := requiredString(m, "title", e)
	if titleOK {
		maxLength(title, "title", 255, e)
	}
	content, _ := optionalString(m, "content", e)

	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	note, err := s.store.CreateNote(r.Context(), u.ID, title, content)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newNoteResponse(note))
}

func (s *Server) showNote(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	note, err := s.store.GetNote(r.Context(), u.ID, r.PathValue("note"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newNoteResponse(note))
}

func (s *Server) updateNote(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	note, err := s.store.GetNote(r.Context(), u.ID, r.PathValue("note"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}

	m := readBody(r)
	e := validationErrors{}

	if v, present := m["title"]; present {
		switch {
		case v == nil:
			e.add("title", "The title field is required.")
		default:
			ts, isStr := v.(string)
			switch {
			case !isStr:
				e.add("title", "The title field must be a string.")
			case ts == "":
				e.add("title", "The title field is required.")
			case utf8.RuneCountInString(ts) > 255:
				e.add("title", "The title field must not be greater than 255 characters.")
			default:
				note.Title = ts
			}
		}
	}

	if v, present := m["content"]; present {
		if v == nil {
			note.Content = ""
		} else if cs, isStr := v.(string); !isStr {
			e.add("content", "The content field must be a string.")
		} else {
			note.Content = cs
		}
	}

	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	updated, err := s.store.UpdateNote(r.Context(), u.ID, note)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newNoteResponse(updated))
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	err := s.store.DeleteNote(r.Context(), u.ID, r.PathValue("note"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
