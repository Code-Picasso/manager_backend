package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"manager-backend/internal/models"
)

func TestCreateTask(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/tasks", token, validTaskPayload(nil))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "title") != "Write tests" {
		t.Fatalf("wrong title: %s", body)
	}
	if jsonCount(t, jsonBytes(t, body, "sub_tasks")) != 2 {
		t.Fatalf("wrong sub-task count: %s", body)
	}
}

func TestCreateTaskValidatesCategoryAndStatus(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/tasks", token, validTaskPayload(map[string]any{"category": "nope"}))
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("category status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["category"]; !ok {
		t.Fatalf("missing category error: %s", body)
	}

	status, body = env.do("POST", "/api/tasks", token, validTaskPayload(map[string]any{"status": "nope"}))
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["status"]; !ok {
		t.Fatalf("missing status error: %s", body)
	}
}

func TestListOwnTasks(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()

	for i := 0; i < 3; i++ {
		env.storeTask(u.ID, models.Task{Title: "Task"})
	}

	status, body := env.do("GET", "/api/tasks", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 3 {
		t.Fatalf("wrong count: %s", body)
	}
}

func TestCannotSeeAnotherUsersTask(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()
	other, _ := env.createUser()
	task := env.storeTask(other.ID, models.Task{Title: "Other task"})

	status, _ := env.do("GET", "/api/tasks/"+task.ID, token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestUpdateTask(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{Title: "Before"})

	status, body := env.do("PUT", "/api/tasks/"+task.ID, token, validTaskPayload(map[string]any{"title": "Updated"}))
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "title") != "Updated" {
		t.Fatalf("wrong title: %s", body)
	}
}

func TestCreateWithoutDescription(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	payload := validTaskPayload(nil)
	delete(payload, "description")

	status, body := env.do("POST", "/api/tasks", token, payload)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "description") != "" {
		t.Fatalf("description not empty: %s", body)
	}
}

func TestDescriptionCanBeCleared(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{Title: "Task", Description: "Something"})

	status, body := env.do("PUT", "/api/tasks/"+task.ID, token, validTaskPayload(map[string]any{"description": ""}))
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "description") != "" {
		t.Fatalf("description not cleared: %s", body)
	}
}

func TestExplicitNullDescriptionStoredEmpty(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/tasks", token, validTaskPayload(map[string]any{"description": nil}))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "description") != "" {
		t.Fatalf("description not empty: %s", body)
	}
}

func TestTaskWithNoSubTasks(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()

	status, body := env.do("POST", "/api/tasks", token, validTaskPayload(map[string]any{"sub_tasks": []any{}}))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, jsonBytes(t, body, "sub_tasks")) != 0 {
		t.Fatalf("sub_tasks not empty: %s", body)
	}
}

func TestDeleteSoftDeletes(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{
		Title:    "Task",
		SubTasks: []models.SubTask{{Title: "A"}},
	})

	status, _ := env.do("DELETE", "/api/tasks/"+task.ID, token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete status = %d", status)
	}

	// Hidden from the list, with its sub-task intact.
	if status, body := env.do("GET", "/api/tasks", token, nil); status != http.StatusOK || jsonCount(t, body) != 0 {
		t.Fatalf("list after delete = %d %s", status, body)
	}
	var subCount int
	if err := testPool.QueryRow(context.Background(), `SELECT count(*) FROM sub_tasks`).Scan(&subCount); err != nil {
		t.Fatal(err)
	}
	if subCount != 1 {
		t.Fatalf("sub-task count = %d, want 1", subCount)
	}
}

func TestRestoreDeletedTask(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{
		Title:    "Task",
		SubTasks: []models.SubTask{{Title: "A"}},
	})

	if status, _ := env.do("DELETE", "/api/tasks/"+task.ID, token, nil); status != http.StatusNoContent {
		t.Fatalf("delete status = %d", status)
	}

	status, body := env.do("POST", "/api/tasks/"+task.ID+"/restore", token, nil)
	if status != http.StatusOK {
		t.Fatalf("restore status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "id") != task.ID {
		t.Fatalf("wrong id: %s", body)
	}
	if jsonCount(t, jsonBytes(t, body, "sub_tasks")) != 1 {
		t.Fatalf("wrong sub-task count: %s", body)
	}

	if status, body := env.do("GET", "/api/tasks", token, nil); status != http.StatusOK || jsonCount(t, body) != 1 {
		t.Fatalf("list after restore = %d %s", status, body)
	}
}

func TestCannotRestoreAnotherUsersTask(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()
	other, _ := env.createUser()
	task := env.storeTask(other.ID, models.Task{Title: "Other task"})

	if err := env.store.SoftDeleteTask(context.Background(), other.ID, task.ID); err != nil {
		t.Fatal(err)
	}

	status, _ := env.do("POST", "/api/tasks/"+task.ID+"/restore", token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestTickingLastSubtaskCompletes(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{
		Title:  "Ticking task",
		Status: "today",
		SubTasks: []models.SubTask{
			{Title: "A", IsDone: false},
			{Title: "B", IsDone: false},
		},
	})

	status, _ := env.do("POST", "/api/tasks/"+task.ID+"/subtasks/"+task.SubTasks[0].ID+"/toggle", token, nil)
	if status != http.StatusOK {
		t.Fatalf("toggle A status = %d", status)
	}

	status, body := env.do("POST", "/api/tasks/"+task.ID+"/subtasks/"+task.SubTasks[1].ID+"/toggle", token, nil)
	if status != http.StatusOK {
		t.Fatalf("toggle B status = %d", status)
	}
	if jsonPath(t, body, "status") != "completed" {
		t.Fatalf("status not completed: %s", body)
	}
}

func TestUntickingSendsCompletedBackToToday(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{
		Title:    "Completed task",
		Status:   "completed",
		Date:     todayStr(),
		SubTasks: []models.SubTask{{Title: "A", IsDone: true}},
	})

	status, body := env.do("POST", "/api/tasks/"+task.ID+"/subtasks/"+task.SubTasks[0].ID+"/toggle", token, nil)
	if status != http.StatusOK {
		t.Fatalf("toggle status = %d", status)
	}
	if jsonPath(t, body, "status") != "today" {
		t.Fatalf("status not today: %s", body)
	}
}

func TestTogglingAnotherUsersSubtaskFails(t *testing.T) {
	env := newTestEnv(t)
	_, token := env.createUser()
	other, _ := env.createUser()
	task := env.storeTask(other.ID, models.Task{
		Title:    "Other task",
		SubTasks: []models.SubTask{{Title: "A"}},
	})

	status, _ := env.do("POST", "/api/tasks/"+task.ID+"/subtasks/"+task.SubTasks[0].ID+"/toggle", token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestFilterByStatus(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()

	env.storeTask(u.ID, models.Task{Title: "Today", Status: "today"})
	env.storeTask(u.ID, models.Task{Title: "Completed", Status: "completed"})
	env.storeTask(u.ID, models.Task{Title: "Coming up", Status: "comingUp"})

	if status, body := env.do("GET", "/api/tasks", token, nil); status != http.StatusOK || jsonCount(t, body) != 3 {
		t.Fatalf("unfiltered = %d %s", status, body)
	}
	if status, body := env.do("GET", "/api/tasks?status=completed", token, nil); status != http.StatusOK || jsonCount(t, body) != 1 {
		t.Fatalf("completed = %d %s", status, body)
	}
	if status, body := env.do("GET", "/api/tasks?status=comingUp", token, nil); status != http.StatusOK || jsonCount(t, body) != 1 {
		t.Fatalf("comingUp = %d %s", status, body)
	}

	status, body := env.do("GET", "/api/tasks?status=nope", token, nil)
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status = %d, body = %s", status, body)
	}
	if _, ok := jsonErrors(t, body)["status"]; !ok {
		t.Fatalf("missing status error: %s", body)
	}
}
