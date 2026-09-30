package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"manager-backend/internal/alerts"
	"manager-backend/internal/models"
)

// ErrNotFound is returned when a row is missing or owned by another user.
var ErrNotFound = errors.New("not found")

// Store owns all database access for the API.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateUser inserts a user and returns it.
func (s *Store) CreateUser(ctx context.Context, name, passwordHash string) (models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (name, password) VALUES ($1, $2) RETURNING id, name`,
		name, passwordHash,
	).Scan(&u.ID, &u.Name)
	return u, err
}

// FindUserByName returns a user by name, or ErrNotFound.
func (s *Store) FindUserByName(ctx context.Context, name string) (models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, password FROM users WHERE name = $1`,
		name,
	).Scan(&u.ID, &u.Name, &u.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// FindUserByID returns a user by id, or ErrNotFound.
func (s *Store) FindUserByID(ctx context.Context, id int64) (models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, name FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// UpdateUserName updates a user's display name and returns the user.
func (s *Store) UpdateUserName(ctx context.Context, id int64, name string) (models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`UPDATE users SET name = $2, updated_at = now() WHERE id = $1 RETURNING id, name`,
		id, name,
	).Scan(&u.ID, &u.Name)
	return u, err
}

// UpdateUserPassword replaces a user's password hash.
func (s *Store) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET password = $2, updated_at = now() WHERE id = $1`,
		id, passwordHash,
	)
	return err
}

// DeleteUser removes a user and, by cascade, their data and tokens.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// CreateToken stores a token hash.
func (s *Store) CreateToken(ctx context.Context, userID int64, tokenHash string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO personal_access_tokens (user_id, token) VALUES ($1, $2)`,
		userID, tokenHash,
	)
	return err
}

// UserForToken resolves a token hash to its owning user, or ErrNotFound.
func (s *Store) UserForToken(ctx context.Context, tokenHash string) (models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.name, u.password FROM personal_access_tokens t JOIN users u ON u.id = t.user_id WHERE t.token = $1`,
		tokenHash,
	).Scan(&u.ID, &u.Name, &u.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// DeleteToken revokes a single token.
func (s *Store) DeleteToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM personal_access_tokens WHERE token = $1`, tokenHash)
	return err
}

// DeleteOtherTokens revokes every token for a user except the given one.
func (s *Store) DeleteOtherTokens(ctx context.Context, userID int64, keepTokenHash string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM personal_access_tokens WHERE user_id = $1 AND token <> $2`,
		userID, keepTokenHash,
	)
	return err
}

// ListTasks returns the user's tasks with sub-tasks, oldest first.
func (s *Store) ListTasks(ctx context.Context, userID int64, status *string) ([]models.Task, error) {
	query := `SELECT id, title, description, category, status, date::text, start_time, end_time, created_at, updated_at
	          FROM tasks WHERE user_id = $1 AND deleted_at IS NULL`
	args := []any{userID}
	if status != nil {
		query += ` AND status = $2`
		args = append(args, *status)
	}
	query += ` ORDER BY date, created_at`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Category, &t.Status, &t.Date, &t.StartTime, &t.EndTime, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.UserID = userID
		tasks = append(tasks, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	subs, err := s.subTasksByTask(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].SubTasks = subs[tasks[i].ID]
	}
	return tasks, nil
}

// GetTask returns one task owned by the user, or ErrNotFound.
func (s *Store) GetTask(ctx context.Context, userID int64, taskID string) (models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, description, category, status, date::text, start_time, end_time, created_at, updated_at
		 FROM tasks WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		taskID, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Category, &t.Status, &t.Date, &t.StartTime, &t.EndTime, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	t.UserID = userID
	subs, err := s.subTasksByTask(ctx, []string{taskID})
	if err != nil {
		return t, err
	}
	t.SubTasks = subs[taskID]
	return t, nil
}

// CreateTask inserts a task and its sub-tasks in one transaction.
func (s *Store) CreateTask(ctx context.Context, task models.Task) (models.Task, error) {
	task.ID = uuid.NewString()
	task.WithAutoStatus(time.Now().UTC())

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return task, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO tasks (id, user_id, title, description, category, status, date, start_time, end_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9)
		 RETURNING created_at, updated_at`,
		task.ID, task.UserID, task.Title, task.Description, task.Category, task.Status, task.Date, task.StartTime, task.EndTime,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return task, err
	}

	for i := range task.SubTasks {
		st := &task.SubTasks[i]
		st.ID = uuid.NewString()
		st.TaskID = task.ID
		st.Position = i
		if _, err := tx.Exec(ctx,
			`INSERT INTO sub_tasks (id, task_id, title, is_done, position) VALUES ($1, $2, $3, $4, $5)`,
			st.ID, st.TaskID, st.Title, st.IsDone, st.Position,
		); err != nil {
			return task, err
		}
	}

	return task, tx.Commit(ctx)
}

// UpdateTask replaces a task and its sub-task list in one transaction.
func (s *Store) UpdateTask(ctx context.Context, task models.Task) (models.Task, error) {
	task.WithAutoStatus(time.Now().UTC())

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return task, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`UPDATE tasks SET title = $2, description = $3, category = $4, status = $5, date = $6::date, start_time = $7, end_time = $8, updated_at = now()
		 WHERE id = $1 AND user_id = $9 AND deleted_at IS NULL
		 RETURNING created_at, updated_at`,
		task.ID, task.Title, task.Description, task.Category, task.Status, task.Date, task.StartTime, task.EndTime, task.UserID,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return task, ErrNotFound
	}
	if err != nil {
		return task, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sub_tasks WHERE task_id = $1`, task.ID); err != nil {
		return task, err
	}
	for i := range task.SubTasks {
		st := &task.SubTasks[i]
		st.ID = uuid.NewString()
		st.TaskID = task.ID
		st.Position = i
		if _, err := tx.Exec(ctx,
			`INSERT INTO sub_tasks (id, task_id, title, is_done, position) VALUES ($1, $2, $3, $4, $5)`,
			st.ID, st.TaskID, st.Title, st.IsDone, st.Position,
		); err != nil {
			return task, err
		}
	}

	return task, tx.Commit(ctx)
}

// SoftDeleteTask marks a task as deleted, or ErrNotFound.
func (s *Store) SoftDeleteTask(ctx context.Context, userID int64, taskID string) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE tasks SET deleted_at = now(), updated_at = now() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		taskID, userID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RestoreTask clears a task's deleted flag and returns it.
func (s *Store) RestoreTask(ctx context.Context, userID int64, taskID string) (models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx,
		`UPDATE tasks SET deleted_at = NULL, updated_at = now()
		 WHERE id = $1 AND user_id = $2
		 RETURNING id, title, description, category, status, date::text, start_time, end_time, created_at, updated_at`,
		taskID, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Category, &t.Status, &t.Date, &t.StartTime, &t.EndTime, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	t.UserID = userID
	subs, err := s.subTasksByTask(ctx, []string{taskID})
	if err != nil {
		return t, err
	}
	t.SubTasks = subs[taskID]
	return t, nil
}

// ToggleSubTask flips one sub-task's done state and re-derives the parent status.
func (s *Store) ToggleSubTask(ctx context.Context, userID int64, taskID, subTaskID string) (models.Task, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE sub_tasks st SET is_done = NOT is_done, updated_at = now()
		 FROM tasks t
		 WHERE st.id = $1 AND st.task_id = t.id AND t.id = $2 AND t.user_id = $3 AND t.deleted_at IS NULL`,
		subTaskID, taskID, userID,
	)
	if err != nil {
		return models.Task{}, err
	}
	if ct.RowsAffected() == 0 {
		return models.Task{}, ErrNotFound
	}

	task, err := s.GetTask(ctx, userID, taskID)
	if err != nil {
		return task, err
	}
	task.WithAutoStatus(time.Now().UTC())
	if _, err := s.pool.Exec(ctx, `UPDATE tasks SET status = $2, updated_at = now() WHERE id = $1`, task.ID, task.Status); err != nil {
		return task, err
	}
	return task, nil
}

// subTasksByTask loads sub-tasks for a set of task ids, grouped by task.
func (s *Store) subTasksByTask(ctx context.Context, taskIDs []string) (map[string][]models.SubTask, error) {
	result := make(map[string][]models.SubTask)
	if len(taskIDs) == 0 {
		return result, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, task_id::text, title, is_done, position FROM sub_tasks WHERE task_id::text = ANY($1) ORDER BY task_id::text, position`,
		taskIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var st models.SubTask
		if err := rows.Scan(&st.ID, &st.TaskID, &st.Title, &st.IsDone, &st.Position); err != nil {
			return nil, err
		}
		result[st.TaskID] = append(result[st.TaskID], st)
	}
	return result, rows.Err()
}

// ListNotes returns the user's notes, most recently updated first.
func (s *Store) ListNotes(ctx context.Context, userID int64) ([]models.Note, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, content, created_at, updated_at FROM notes WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]models.Note, 0)
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		n.UserID = userID
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// CreateNote inserts a note and returns it.
func (s *Store) CreateNote(ctx context.Context, userID int64, title, content string) (models.Note, error) {
	var n models.Note
	err := s.pool.QueryRow(ctx,
		`INSERT INTO notes (id, user_id, title, content) VALUES ($1, $2, $3, $4) RETURNING id, title, content, created_at, updated_at`,
		uuid.NewString(), userID, title, content,
	).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	n.UserID = userID
	return n, err
}

// GetNote returns one note owned by the user, or ErrNotFound.
func (s *Store) GetNote(ctx context.Context, userID int64, noteID string) (models.Note, error) {
	var n models.Note
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, content, created_at, updated_at FROM notes WHERE id = $1 AND user_id = $2`,
		noteID, userID,
	).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return n, ErrNotFound
	}
	if err != nil {
		return n, err
	}
	n.UserID = userID
	return n, nil
}

// UpdateNote updates a note and returns it, or ErrNotFound.
func (s *Store) UpdateNote(ctx context.Context, userID int64, note models.Note) (models.Note, error) {
	err := s.pool.QueryRow(ctx,
		`UPDATE notes SET title = $3, content = $4, updated_at = now() WHERE id = $1 AND user_id = $2 RETURNING id, title, content, created_at, updated_at`,
		note.ID, userID, note.Title, note.Content,
	).Scan(&note.ID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return note, ErrNotFound
	}
	if err != nil {
		return note, err
	}
	note.UserID = userID
	return note, nil
}

// DeleteNote removes a note, or ErrNotFound.
func (s *Store) DeleteNote(ctx context.Context, userID int64, noteID string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1 AND user_id = $2`, noteID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAlerts returns the user's alerts, newest first.
func (s *Store) ListAlerts(ctx context.Context, userID int64) ([]models.Alert, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, message, is_success, "group", is_read, created_at FROM alerts WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Alert, 0)
	for rows.Next() {
		var a models.Alert
		if err := rows.Scan(&a.ID, &a.Message, &a.IsSuccess, &a.Group, &a.IsRead, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.UserID = userID
		result = append(result, a)
	}
	return result, rows.Err()
}

// MarkAllRead flags every unread alert as read.
func (s *Store) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE alerts SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`, userID)
	return err
}

// SyncAlerts reconciles stored alerts with the set implied by the user's tasks.
func (s *Store) SyncAlerts(ctx context.Context, userID int64, now time.Time) error {
	tasks, err := s.tasksForAlerts(ctx, userID)
	if err != nil {
		return err
	}
	desired := alerts.Desired(tasks, now)

	keys := make([]string, 0, len(desired))
	for _, d := range desired {
		keys = append(keys, d.Key)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`DELETE FROM alerts WHERE user_id = $1
		 AND (key LIKE 'task-completed-%' OR key LIKE 'task-due-soon-%' OR key LIKE 'task-overdue-%')
		 AND NOT (key = ANY($2))`,
		userID, keys,
	)
	if err != nil {
		return err
	}

	for _, d := range desired {
		if _, err := tx.Exec(ctx,
			`INSERT INTO alerts (id, user_id, key, message, is_success, "group", is_read)
			 VALUES ($1, $2, $3, $4, $5, 'Today', FALSE)
			 ON CONFLICT (key) DO NOTHING`,
			uuid.NewString(), userID, d.Key, d.Message, d.IsSuccess,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// tasksForAlerts loads the fields the alert scan needs.
func (s *Store) tasksForAlerts(ctx context.Context, userID int64) ([]models.Task, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, status, date::text, end_time FROM tasks WHERE user_id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Date, &t.EndTime); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}
