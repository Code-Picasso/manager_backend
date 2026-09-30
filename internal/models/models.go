package models

import (
	"strconv"
	"strings"
	"time"
)

// User is an account owner.
type User struct {
	ID       int64
	Name     string
	Email    string
	Password string
}

// Task is a dated to-do with an optional checklist of sub-tasks.
type Task struct {
	ID          string
	UserID      int64
	Title       string
	Description string
	Category    string
	Status      string
	Date        string
	StartTime   string
	EndTime     string
	SubTasks    []SubTask
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SubTask is a checklist item inside a task.
type SubTask struct {
	ID       string
	TaskID   string
	Title    string
	IsDone   bool
	Position int
}

// Note is a free-form note owned by a user.
type Note struct {
	ID        string
	UserID    int64
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Alert is a notification derived from task state.
type Alert struct {
	ID        string
	UserID    int64
	Key       string
	Message   string
	IsSuccess bool
	Group     string
	IsRead    bool
	CreatedAt time.Time
}

// Progress returns the completion ratio between 0 and 1.
func (t Task) Progress() float64 {
	if len(t.SubTasks) == 0 {
		if t.Status == "completed" {
			return 1
		}
		return 0
	}
	done := 0
	for _, s := range t.SubTasks {
		if s.IsDone {
			done++
		}
	}
	return float64(done) / float64(len(t.SubTasks))
}

func (t Task) allSubTasksDone() bool {
	if len(t.SubTasks) == 0 {
		return false
	}
	for _, s := range t.SubTasks {
		if !s.IsDone {
			return false
		}
	}
	return true
}

// WithAutoStatus re-derives the status from sub-task progress and the due date.
func (t *Task) WithAutoStatus(now time.Time) {
	if len(t.SubTasks) == 0 {
		return
	}
	if t.allSubTasksDone() {
		if t.Status != "completed" {
			t.Status = "completed"
		}
		return
	}
	if t.Status != "completed" {
		return
	}
	due, err := time.Parse("2006-01-02", t.Date)
	if err != nil {
		return
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if due.After(today) {
		t.Status = "comingUp"
	} else {
		t.Status = "today"
	}
}

// Deadline is the moment the task is due — its date at its end time.
func (t Task) Deadline() time.Time {
	d, err := time.Parse("2006-01-02", t.Date)
	if err != nil {
		d = time.Time{}
	}
	hour, minute := parseTime(t.EndTime)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, time.UTC)
}

// parseTime splits "HH:MM", defaulting to 23:59 for malformed input.
func parseTime(value string) (hour, minute int) {
	hour, minute = 23, 59
	parts := strings.SplitN(value, ":", 2)
	if len(parts) > 0 {
		if h, err := strconv.Atoi(parts[0]); err == nil {
			hour = h
		}
	}
	if len(parts) > 1 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			minute = m
		}
	}
	return hour, minute
}
