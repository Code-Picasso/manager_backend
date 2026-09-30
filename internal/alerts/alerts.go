package alerts

import (
	"fmt"
	"time"

	"manager-backend/internal/models"
)

const (
	completedPrefix = "task-completed-"
	dueSoonPrefix   = "task-due-soon-"
	overduePrefix   = "task-overdue-"
)

// Spec is a single alert implied by the current task state.
type Spec struct {
	Key       string
	Message   string
	IsSuccess bool
}

// Desired derives the alerts the current task list implies.
func Desired(tasks []models.Task, now time.Time) []Spec {
	var result []Spec

	for _, task := range tasks {
		if task.Status == "completed" {
			result = append(result, Spec{
				Key:       completedPrefix + task.ID,
				Message:   fmt.Sprintf("Nice work — you finished \"%s\".", task.Title),
				IsSuccess: true,
			})
			continue
		}

		deadline := task.Deadline()

		if now.After(deadline) {
			if now.Sub(deadline) > 7*24*time.Hour {
				continue
			}
			result = append(result, Spec{
				Key:       overduePrefix + task.ID,
				Message:   fmt.Sprintf("\"%s\" was due %s at %s.", task.Title, mediumDate(task.Date), task.EndTime),
				IsSuccess: false,
			})
			continue
		}

		if deadline.Sub(now) <= time.Hour {
			result = append(result, Spec{
				Key:       dueSoonPrefix + task.ID,
				Message:   fmt.Sprintf("Time is running out — \"%s\" is due at %s.", task.Title, task.EndTime),
				IsSuccess: false,
			})
		}
	}

	return result
}

var shortMonths = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// mediumDate formats a "YYYY-MM-DD" date as "Oct 24th, 2023".
func mediumDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return fmt.Sprintf("%s %s, %d", shortMonths[t.Month()-1], ordinal(t.Day()), t.Year())
}

// ordinal returns a day with its English ordinal suffix.
func ordinal(day int) string {
	if day >= 11 && day <= 13 {
		return fmt.Sprintf("%dth", day)
	}
	switch day % 10 {
	case 1:
		return fmt.Sprintf("%dst", day)
	case 2:
		return fmt.Sprintf("%dnd", day)
	case 3:
		return fmt.Sprintf("%drd", day)
	default:
		return fmt.Sprintf("%dth", day)
	}
}
