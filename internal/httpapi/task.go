package httpapi

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"manager-backend/internal/models"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)

	var status *string
	if r.URL.Query().Has("status") {
		q := r.URL.Query().Get("status")
		if q != "today" && q != "comingUp" && q != "completed" {
			writeValidationError(w, validationErrors{"status": {"The selected status is invalid."}})
			return
		}
		status = &q
	}

	tasks, err := s.store.ListTasks(r.Context(), u.ID, status)
	if err != nil {
		serverError(w, err)
		return
	}

	resp := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, newTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	m := readBody(r)
	e := validationErrors{}
	t := parseTaskBody(m, e)
	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	t.UserID = u.ID
	created, err := s.store.CreateTask(r.Context(), t)
	if err != nil {
		serverError(w, err)
		return
	}
	s.syncAlerts(r.Context(), u.ID)

	writeJSON(w, http.StatusCreated, newTaskResponse(created))
}

func (s *Server) showTask(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	task, err := s.store.GetTask(r.Context(), u.ID, r.PathValue("task"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	m := readBody(r)
	e := validationErrors{}
	t := parseTaskBody(m, e)
	if len(e) > 0 {
		writeValidationError(w, e)
		return
	}

	t.ID = r.PathValue("task")
	t.UserID = u.ID

	updated, err := s.store.UpdateTask(r.Context(), t)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	s.syncAlerts(r.Context(), u.ID)

	writeJSON(w, http.StatusOK, newTaskResponse(updated))
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	err := s.store.SoftDeleteTask(r.Context(), u.ID, r.PathValue("task"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	s.syncAlerts(r.Context(), u.ID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreTask(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	task, err := s.store.RestoreTask(r.Context(), u.ID, r.PathValue("task"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	s.syncAlerts(r.Context(), u.ID)

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

func (s *Server) toggleSubTask(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	task, err := s.store.ToggleSubTask(r.Context(), u.ID, r.PathValue("task"), r.PathValue("subTask"))
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		serverError(w, err)
		return
	}
	s.syncAlerts(r.Context(), u.ID)

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// parseTaskBody validates a create/update task payload and returns the task.
func parseTaskBody(m map[string]any, e validationErrors) models.Task {
	var t models.Task

	if title, ok := requiredString(m, "title", e); ok {
		maxLength(title, "title", 255, e)
		t.Title = title
	}

	t.Description, _ = optionalString(m, "description", e)
	t.Category = requiredEnum(m, "category", []string{"work", "personal"}, e)
	t.Status = requiredEnum(m, "status", []string{"today", "comingUp", "completed"}, e)
	t.Date = requiredDate(m, "date", e)
	t.StartTime = requiredTime(m, "start_time", e)
	t.EndTime = requiredTime(m, "end_time", e)

	if raw, ok := m["sub_tasks"]; ok && raw != nil {
		arr, isArr := raw.([]any)
		if !isArr {
			e.add("sub_tasks", "The sub_tasks field must be an array.")
			return t
		}
		for i, item := range arr {
			prefix := "sub_tasks." + strconv.Itoa(i)
			var st models.SubTask

			obj, isMap := item.(map[string]any)
			if !isMap {
				e.add(prefix+".title", "The "+prefix+".title field is required.")
				continue
			}

			if tv, ok := obj["title"]; !ok || tv == nil {
				e.add(prefix+".title", "The "+prefix+".title field is required.")
			} else if ts, isStr := tv.(string); !isStr {
				e.add(prefix+".title", "The "+prefix+".title field must be a string.")
			} else if ts == "" {
				e.add(prefix+".title", "The "+prefix+".title field is required.")
			} else if utf8.RuneCountInString(ts) > 255 {
				e.add(prefix+".title", "The "+prefix+".title field must not be greater than 255 characters.")
			} else {
				st.Title = ts
			}

			if dv, ok := obj["is_done"]; ok && dv != nil {
				if b, isBool := dv.(bool); isBool {
					st.IsDone = b
				} else {
					e.add(prefix+".is_done", "The "+prefix+".is_done field must be true or false.")
				}
			}

			t.SubTasks = append(t.SubTasks, st)
		}
	}

	return t
}
