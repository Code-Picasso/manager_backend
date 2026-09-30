package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"manager-backend/internal/models"
)

func TestCompletedTaskProducesCongratulation(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	env.storeTask(u.ID, models.Task{Title: "Ship the API", Status: "completed"})

	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 1 {
		t.Fatalf("wrong count: %s", body)
	}
	if jsonPath(t, body, "0", "is_success") != true {
		t.Fatalf("is_success not true: %s", body)
	}
	if jsonPath(t, body, "0", "message") != "Nice work — you finished \"Ship the API\"." {
		t.Fatalf("wrong message: %s", body)
	}
}

func TestOverdueTaskProducesWarning(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	env.storeTask(u.ID, models.Task{Title: "Overdue", Status: "today", Date: yesterdayStr(), EndTime: "10:00"})

	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 1 {
		t.Fatalf("wrong count: %s", body)
	}
	if jsonPath(t, body, "0", "is_success") != false {
		t.Fatalf("is_success not false: %s", body)
	}
}

func TestReconciliationNeverDuplicates(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	env.storeTask(u.ID, models.Task{Title: "Done", Status: "completed"})

	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 1 {
		t.Fatalf("duplicates found: %s", body)
	}
}

func TestReconciliationRemovesStaleAlerts(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	task := env.storeTask(u.ID, models.Task{Title: "Done", Status: "completed"})

	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if err := env.store.SoftDeleteTask(t.Context(), u.ID, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 0 {
		t.Fatalf("stale alert remains: %s", body)
	}
}

func TestMarkAllRead(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	env.storeTask(u.ID, models.Task{Title: "Done", Status: "completed"})

	if err := env.store.SyncAlerts(t.Context(), u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if status, _ := env.do("POST", "/api/alerts/read-all", token, nil); status != http.StatusOK {
		t.Fatalf("read-all status = %d", status)
	}

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonPath(t, body, "0", "is_read") != true {
		t.Fatalf("alert not read: %s", body)
	}
}

func TestListingReconcilesWithoutTaskChange(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	env.storeTask(u.ID, models.Task{Title: "Overdue", Status: "today", Date: yesterdayStr(), EndTime: "10:00"})

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 1 {
		t.Fatalf("wrong count: %s", body)
	}
	if jsonPath(t, body, "0", "is_success") != false {
		t.Fatalf("is_success not false: %s", body)
	}
}

func TestListingNeverResurrectsReadAlert(t *testing.T) {
	env := newTestEnv(t)
	u, token := env.createUser()
	env.storeTask(u.ID, models.Task{Title: "Done", Status: "completed"})

	if status, body := env.do("GET", "/api/alerts", token, nil); status != http.StatusOK || jsonCount(t, body) != 1 {
		t.Fatalf("first read = %d %s", status, body)
	}
	if status, _ := env.do("POST", "/api/alerts/read-all", token, nil); status != http.StatusOK {
		t.Fatalf("read-all status = %d", status)
	}

	status, body := env.do("GET", "/api/alerts", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if jsonCount(t, body) != 1 {
		t.Fatalf("wrong count: %s", body)
	}
	if jsonPath(t, body, "0", "is_read") != true {
		t.Fatalf("alert not read: %s", body)
	}
}
