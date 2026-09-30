package httpapi

import (
	"encoding/json"
	"time"

	"manager-backend/internal/models"
)

// apiTime marshals a timestamp as Laravel's ISO-8601 UTC shape.
type apiTime time.Time

func (t apiTime) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(time.Time(t).UTC().Format("2006-01-02T15:04:05-07:00"))
}

type userResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type subTaskResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	IsDone bool   `json:"is_done"`
}

type taskResponse struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Status      string            `json:"status"`
	Date        string            `json:"date"`
	StartTime   string            `json:"start_time"`
	EndTime     string            `json:"end_time"`
	SubTasks    []subTaskResponse `json:"sub_tasks"`
	Progress    float64           `json:"progress"`
	CreatedAt   apiTime           `json:"created_at"`
	UpdatedAt   apiTime           `json:"updated_at"`
}

type noteResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	CreatedAt apiTime `json:"created_at"`
	UpdatedAt apiTime `json:"updated_at"`
}

type alertResponse struct {
	ID        string  `json:"id"`
	Message   string  `json:"message"`
	IsSuccess bool    `json:"is_success"`
	Group     string  `json:"group"`
	IsRead    bool    `json:"is_read"`
	CreatedAt apiTime `json:"created_at"`
}

func newUserResponse(u models.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name}
}

func newTaskResponse(t models.Task) taskResponse {
	subs := make([]subTaskResponse, 0, len(t.SubTasks))
	for _, st := range t.SubTasks {
		subs = append(subs, subTaskResponse{ID: st.ID, Title: st.Title, IsDone: st.IsDone})
	}
	return taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Category:    t.Category,
		Status:      t.Status,
		Date:        t.Date,
		StartTime:   t.StartTime,
		EndTime:     t.EndTime,
		SubTasks:    subs,
		Progress:    t.Progress(),
		CreatedAt:   apiTime(t.CreatedAt),
		UpdatedAt:   apiTime(t.UpdatedAt),
	}
}

func newNoteResponse(n models.Note) noteResponse {
	return noteResponse{
		ID:        n.ID,
		Title:     n.Title,
		Content:   n.Content,
		CreatedAt: apiTime(n.CreatedAt),
		UpdatedAt: apiTime(n.UpdatedAt),
	}
}

func newAlertResponse(a models.Alert) alertResponse {
	return alertResponse{
		ID:        a.ID,
		Message:   a.Message,
		IsSuccess: a.IsSuccess,
		Group:     a.Group,
		IsRead:    a.IsRead,
		CreatedAt: apiTime(a.CreatedAt),
	}
}
