# M.N.G.R Manager — Laravel Backend

A **Laravel 12 + MySQL + Docker** REST API that mirrors the domain of the
Flutter [M.N.G.R Manager](../manager) app: users, tasks/sub-tasks, notes, and
derived alerts.

This project exists to **teach Laravel**. The code is heavily commented, and
this README walks through every concept it uses — from the request lifecycle to
Eloquent relationships. Read it top-to-bottom once, then re-read the code and
everything will click.

---

## Table of contents

1. [What this is](#what-this-is)
2. [Quick start (Docker)](#quick-start-docker)
3. [Quick start (local, no Docker)](#quick-start-local-no-docker)
4. [The request lifecycle](#the-request-lifecycle)
5. [Project structure](#project-structure)
6. [Core concepts, explained](#core-concepts-explained)
7. [API reference](#api-reference)
8. [Mapping to the Flutter app](#mapping-to-the-flutter-app)
9. [Password hashing: bcrypt vs PBKDF2](#password-hashing-bcrypt-vs-pbkdf2)
10. [Alerts: derived, not stored](#alerts-derived-not-stored)
11. [Testing](#testing)
12. [Going to production](#going-to-production)

---

## What this is

The Flutter app is local-first: everything lives on-device in Hive. This backend
re-implements the *same* domain server-side so the app could later be pointed at
a real API instead of local storage. The data model and business rules match the
app exactly:

| Concept               | Flutter app                          | This backend                              |
| --------------------- | ------------------------------------ | ----------------------------------------- |
| Account / auth        | PBKDF2 hash + session key in Hive    | bcrypt hash + Sanctum bearer token        |
| Task + sub-tasks      | embedded JSON in a Hive box          | `tasks` + `sub_tasks` tables (normalised) |
| Task status           | derived from sub-task state          | `Task::withAutoStatus()` — same rules     |
| Notes                 | Hive box, sorted by `updatedAt`      | `notes` table                             |
| Alerts                | derived + reconciled from tasks      | `TaskAlertsService` — same rules          |

---

## Quick start (Docker)

The easiest way to run everything is with Docker Compose. It starts three
containers: the API (`app`), MySQL (`db`), and phpMyAdmin (a browser UI for the
database).

```bash
cd manager_backend
docker compose up --build
```

On first start the `app` container waits for MySQL, runs the migrations, then
serves the API. You'll have:

| URL                    | What it is                                         |
| ---------------------- | -------------------------------------------------- |
| http://localhost:8000  | The API (`/api/...`)                               |
| http://localhost:8080  | phpMyAdmin (server `db`, user `manager`, password `secret`) |

Quick smoke test:

```bash
# Register a user (returns a token)
curl -X POST http://localhost:8000/api/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada","email":"ada@example.com","password":"Secret123!"}'

# Use the returned token to list tasks
curl http://localhost:8000/api/tasks \
  -H 'Authorization: Bearer <TOKEN>'
```

Stop it with `docker compose down` (add `-v` to also delete the database).

> **Docker daemon not running?** Start Docker Desktop first. On macOS:
> `open -a Docker`, then wait for the whale icon to finish animating.

---

## Quick start (local, no Docker)

You don't need Docker to develop. Use SQLite (a single file) instead of MySQL.

```bash
cd manager_backend

# 1. Install dependencies
composer install

# 2. Configure SQLite (one-time)
cp .env.example .env
#   edit .env and set:  DB_CONNECTION=sqlite   (and remove the MySQL lines)

# 3. Generate an app key and run migrations
php artisan key:generate
php artisan migrate

# 4. Serve
php artisan serve
```

Then hit `http://localhost:8000/api/...` the same way.

---

## The request lifecycle

Before diving into files, understand the one path every request takes. Laravel
routes every HTTP request through the same pipeline:

```
HTTP request
   │
   ▼
public/index.php          ← the single entry point for every request
   │
   ▼
bootstrap/app.php         ← builds the "application" and registers routing + middleware
   │
   ▼
Global middleware         ← rate limiting, CORS, trimming strings, etc.
   │
   ▼
Route matched             ← URL + method -> controller method (routes/api.php)
   │
   ▼
Route middleware          ← e.g. auth:sanctum (checks the Bearer token)
   │
   ▼
Form Request validation   ← invalid? return 422 before the controller runs
   │
   ▼
Controller method         ← your logic: read/write the database, call services
   │
   ▼
Response                  ← a JSON resource, a status code, etc.
```

The key idea: **a controller method should be thin.** Validation lives in a Form
Request, data access lives in Eloquent models, JSON shaping lives in a Resource,
and business rules live in services or model methods. The controller just
orchestrates them.

---

## Project structure

```
manager_backend/
├── app/
│   ├── Enums/                  # TaskCategory, TaskStatus (backed enums)
│   ├── Http/
│   │   ├── Controllers/        # Auth, Task, Note, Alert controllers
│   │   ├── Requests/           # Form Requests (validation)
│   │   └── Resources/          # JSON shaping (User/Task/Note/Alert resources)
│   ├── Models/                 # User, Task, SubTask, Note, Alert
│   ├── Providers/              # AppServiceProvider (boot hooks)
│   └── Services/               # TaskAlertsService (alert reconciliation)
├── bootstrap/
│   ├── app.php                 # application configuration + middleware
│   └── providers.php           # service providers to register
├── config/                     # framework + package config (database, auth, …)
├── database/
│   ├── factories/              # test data generators
│   ├── migrations/             # schema definitions (versioned)
│   └── seeders/                # optional demo-data seeders
├── docker/entrypoint.sh        # waits for MySQL, migrates, serves
├── routes/
│   ├── api.php                 # all /api routes
│   ├── web.php                 # browser routes (unused here)
│   └── console.php             # Artisan command routes
├── tests/                      # feature + unit tests (run against SQLite)
├── Dockerfile
├── docker-compose.yml
└── composer.json               # dependencies + autoloading
```

---

## Core concepts, explained

### MVC — Model, View, Controller

Laravel is an MVC framework. This API has no "View" (it returns JSON), so think
of it as **MC**:

- **Model** — a class that represents a database table and its behaviour
  (`app/Models/Task.php`). One row = one model instance.
- **Controller** — a class whose methods handle requests
  (`app/Http/Controllers/TaskController.php`).

The "View" layer is replaced by **API Resources** (`app/Http/Resources/`), which
transform models into the JSON the client expects.

### Routing

`routes/api.php` maps a URL + HTTP verb to a controller method:

```php
Route::get('/tasks', [TaskController::class, 'index']);
```

`Route::apiResource('tasks', TaskController::class)` is shorthand that expands
into the five standard CRUD routes (`index`, `store`, `show`, `update`,
`destroy`). Run `php artisan route:list` to see the full table.

`Route::middleware('auth:sanctum')->group(...)` wraps routes so that **every
request inside must carry a valid token**. The `auth:sanctum` middleware reads
the `Authorization: Bearer <token>` header and resolves the user — or returns
`401` if the token is missing/invalid.

### Controllers and dependency injection

```php
class TaskController extends Controller
{
    public function __construct(private readonly TaskAlertsService $alerts) {}
    // ...
}
```

This is **constructor injection**. Laravel's *service container* automatically
creates a `TaskAlertsService` and passes it in when the controller is
instantiated. You never write `new TaskAlertsService()` — the container resolves
it (and its own dependencies) for you. This is the heart of Laravel: you ask for
what you need as a type-hint, and the container supplies it.

### Route model binding

```php
public function show(Request $request, Task $task): TaskResource
{
    // $task is already fetched from the DB by its id
}
```

Because the route is `/tasks/{task}`, Laravel sees the `Task $task` type-hint,
looks up the row with that id, and injects it. If no row exists it returns `404`
automatically. No manual `Task::find($id)` needed.

Because this is a multi-user API, every method then checks **ownership**:

```php
abort_unless($task->user_id === $request->user()->id, 404);
```

This returns `404` (not `403`) so a user can't even probe whether another
user's task exists. (The scalable alternative — `$this->authorize('update', $task)`
with a [Policy](https://laravel.com/docs/12.x/authorization) — is described in
[Going to production](#going-to-production).)

### Models & Eloquent

Eloquent is Laravel's **ORM** (Object-Relational Mapper). Each model maps to a
table, and you interact with rows as objects instead of writing SQL:

```php
$request->user()->tasks()->create([...]);   // INSERT
$task->update([...]);                        // UPDATE
$task->delete();                             // DELETE
Task::where('title', 'like', '%x%')->get(); // SELECT
```

Key model features used in this project:

- **`$fillable`** — the list of columns allowed for mass-assignment. Writing to
  a column not listed here via `create()`/`update()` is silently ignored. This
  defends against *mass-assignment* attacks where a client sends extra fields
  (e.g. `is_admin => true`).

- **`$hidden`** — columns never serialised (see `User::$hidden`, which hides
  `password` and `remember_token`).

- **Casts** — declared in `casts()`, they convert a raw DB value into a richer
  PHP type when reading a model. For example `Task::$casts` maps `category` to
  the `TaskCategory` enum and `date` to a `Carbon` date object.

- **Accessors** — a method named `getProgressAttribute()` exposes `$task->progress`
  as if it were a real column, computed on the fly from sub-tasks.

- **`HasUuids`** — makes the primary key a UUID string (matching the Flutter
  app's string ids) instead of an auto-incrementing integer, and generates one
  automatically on create.

### Migrations

`database/migrations/` defines the schema as **versioned PHP**, so the database
can be rebuilt deterministically anywhere (dev, CI, a teammate's machine):

```php
Schema::create('sub_tasks', function (Blueprint $table) {
    $table->uuid('id')->primary();
    $table->foreignUuid('task_id')->constrained()->cascadeOnDelete();
    $table->string('title');
    $table->boolean('is_done')->default(false);
    $table->unsignedInteger('position')->default(0);
    $table->timestamps();
});
```

- `foreignUuid('task_id')->constrained()` creates a foreign key to `tasks.id`.
- `cascadeOnDelete()` means deleting a task auto-deletes its sub-tasks (and the
  same pattern ties users → tasks/notes/alerts).
- `timestamps()` adds `created_at`/`updated_at`, which Eloquent maintains
  automatically.

Run migrations with `php artisan migrate`. In Docker, the entrypoint runs
`php artisan migrate --force` automatically on start.

### Form Requests (validation)

`app/Http/Requests/*.php` centralise validation *outside* the controller:

```php
public function rules(): array
{
    return [
        'title' => ['required', 'string', 'max:255'],
        'category' => ['required', Rule::enum(TaskCategory::class)],
        'start_time' => ['required', 'date_format:H:i'],
        // ...
    ];
}
```

Laravel runs these **before** the controller method executes. If they fail it
returns a `422` JSON response with per-field errors; the controller can assume
the data is valid. `authorize()` gates who may make the request (here it's
`true` because auth is handled by middleware + ownership checks).

Note `Rule::enum(...)` — it rejects any value that isn't a valid enum case, so
`category` can only be `work` or `personal`.

For the password policy, the rule is a **custom rule object**
(`app/Rules/StrongPassword.php`). Instead of repeating the same `regex:` strings
in every request that accepts a password, each one just adds `new StrongPassword`
to its rules. A custom rule is any class implementing `ValidationRule` whose
`validate()` method reports problems through the `$fail` closure.

#### `prepareForValidation()` and empty strings

Laravel's global `ConvertEmptyStringsToNull` middleware rewrites every `""` in the
request to `null` **before** validation. That is a sensible default for genuinely
optional columns — but `tasks.description` and `notes.content` are `NOT NULL`
with an empty-string default, so a `null` here reaches the database and the write
fails with an integrity-constraint violation (a `500`, not a `422`).

The rules on those fields say `nullable`, so a client that sends `""` — which the
Flutter app does for every task saved without a description — is being told the
value is acceptable right up until it isn't.

`app/Http/Requests/Concerns/NormalizesEmptyText.php` closes that gap:

```php
protected function prepareForValidation(): void
{
    $this->normalizeEmptyText(['description']);
}
```

`prepareForValidation()` runs before `rules()`, so the value is a string again by
the time validation and the controller see it. The columns also carry
`protected $attributes = ['description' => ''];` on their models, mirroring the
migration defaults — without that, `POST /tasks` would answer `null` for a task
created without a description while `GET /tasks` answered `""` for the same row.

### API Resources

`app/Http/Resources/*.php` are the "view" layer for JSON. They control exactly
what the client sees, decoupling the database columns from the API shape:

```php
public function toArray(Request $request): array
{
    return [
        'id' => $this->id,
        'category' => $this->category->value,          // enum -> 'work'
        'date' => $this->date->toDateString(),         // Carbon -> 'YYYY-MM-DD'
        'sub_tasks' => SubTaskResource::collection($this->whenLoaded('subTasks')),
    ];
}
```

`$this->whenLoaded('subTasks')` only embeds sub-tasks if the relation was
eager-loaded (see below) — otherwise it's omitted, which also avoids accidental
N+1 queries during serialisation.

### Eloquent relationships

Normalising sub-tasks into their own table means we need relationships to join
them back together:

```php
// Task
public function subTasks(): HasMany { return $this->hasMany(SubTask::class)->orderBy('position'); }

// SubTask
public function task(): BelongsTo { return $this->belongsTo(Task::class); }
```

- `hasMany` / `belongsTo` / (on `User`) `hasMany` define the connections.
- `$task->subTasks` returns the related collection.
- **Eager loading** — `->with('subTasks')` — loads all sub-tasks in one extra
  query up front. Without it, each `$task->subTasks` access would hit the DB
  separately (the "N+1 query" problem). You'll see `with('subTasks')` in the
  index methods.

### Enums

`app/Enums/TaskCategory.php` and `TaskStatus.php` are PHP **backed enums** — each
case carries a string value. Because the model casts the column to the enum, the
DB stores `'work'` / `'today'` while your code works with `TaskCategory::Work` /
`TaskStatus::Today`. The string values deliberately match the Flutter app's enum
names so the JSON is compatible.

### Service providers

`app/Providers/AppServiceProvider.php` is a **service provider** — a bootstrap
hook. Its `boot()` runs after everything is registered and is where we:

1. Disable the JSON `data` wrapper (`JsonResource::withoutWrapping()`), so
   resources return plain objects/arrays like the Flutter app expects.
2. Define the `api` rate limiter (60 requests/minute per user/IP).

`bootstrap/app.php` then applies that limiter to the whole API group with
`$middleware->throttleApi('api')`.

### Sanctum token auth

[Laravel Sanctum](https://laravel.com/docs/12.x/sanctum) provides API token
authentication. The `User` model uses the `HasApiTokens` trait, which adds
`$user->createToken('name')->plainTextToken`. That token is sent by the client
as a bearer header and validated by the `auth:sanctum` middleware.

```php
$token = $user->createToken('access-token')->plainTextToken; // issue
$request->user()->currentAccessToken()->delete();            // revoke (logout)
```

### Transactions

`TaskController::store` wraps task + sub-task creation in `DB::transaction(...)`.
If anything inside throws, **everything** is rolled back — you can never end up
with a task that has half its sub-tasks. (MySQL is transactional; this is the
kind of correctness you only get for free on a real database, unlike Hive.)

### Soft deletes & the Undo button

The app's task-delete dialog offers an **Undo** (`TasksController.restoreTask`)
that re-inserts the exact task it just removed. A hard `DELETE` can't support
that, so `Task` uses Eloquent's **`SoftDeletes`** trait:

- `DELETE /api/tasks/{task}` sets a `deleted_at` timestamp instead of removing
  the row. The task vanishes from every normal query (a global scope filters
  trashed rows), but its sub-tasks are left intact.
- `POST /api/tasks/{task}/restore` clears `deleted_at`, bringing the task back
  with the **same id** and its checklist — which is exactly what the UI's undo
  expects. The route is declared `->withTrashed()` so model binding can find a
  row the default scope hides.

Soft deletion also cleans up alerts for free: a trashed task is excluded from
the alert scan, so its warnings disappear, and they reappear on restore.

> **Note:** Notes are hard-deleted, because the notes UI has no undo. Only tasks
> are restorable.

### Factories & testing

`database/factories/` generate fake models for tests/seeding:

```php
Task::factory()->create(['user_id' => $user->id]);
```

`tests/Feature/*` then exercise the real HTTP endpoints. Tests run against an
in-memory SQLite database (configured in `phpunit.xml`), so no MySQL is needed.
Run them with:

```bash
php artisan test
```

---

## API reference

All endpoints are JSON. Protected routes require `Authorization: Bearer <token>`.

### Auth

| Method | Endpoint               | Body                                         | Auth | Returns                          |
| ------ | ---------------------- | -------------------------------------------- | ---- | -------------------------------- |
| POST   | `/api/register`        | `name, email, password`                      | —    | `{ user, token }` (201)          |
| POST   | `/api/login`           | `email, password`                            | —    | `{ user, token }`                |
| POST   | `/api/reset-password`  | `email, password` (the new one)              | —    | `{ message }`                    |
| GET    | `/api/user`            | —                                            | ✓    | `{ id, name, email }`            |
| PUT    | `/api/user`            | `name`                                       | ✓    | `{ id, name, email }`            |
| DELETE | `/api/user`            | —                                            | ✓    | `{ message }` (deletes account)  |
| POST   | `/api/logout`          | —                                            | ✓    | `{ message }` (revokes token)    |
| POST   | `/api/change-password` | `current_password`, `password` (the new one) | ✓    | `{ message }`                    |

### Tasks

| Method | Endpoint                                        | Purpose                                   |
| ------ | ----------------------------------------------- | ----------------------------------------- |
| GET    | `/api/tasks`                                    | list own tasks (with `sub_tasks`)         |
| POST   | `/api/tasks`                                    | create task + sub-tasks                   |
| GET    | `/api/tasks/{task}`                             | show one task                             |
| PUT    | `/api/tasks/{task}`                             | update (full replace, incl. sub-tasks)    |
| DELETE | `/api/tasks/{task}`                             | delete (soft — restorable)                |
| POST   | `/api/tasks/{task}/restore`                     | undo a delete (restores the task)         |
| POST   | `/api/tasks/{task}/subtasks/{subTask}/toggle`   | flip one sub-task's `is_done`             |

`GET /api/tasks` also accepts an optional `?status=today|comingUp|completed`
filter, so the board tabs can be served server-side. An unknown value returns
`422` rather than an empty list.

Task JSON shape:

```json
{
  "id": "9d1a…",
  "title": "Ship the API",
  "description": "",
  "category": "work",
  "status": "today",
  "date": "2026-09-15",
  "start_time": "09:00",
  "end_time": "10:00",
  "sub_tasks": [
    { "id": "…", "title": "Write controllers", "is_done": false }
  ],
  "progress": 0,
  "created_at": "2026-09-15T12:00:00+00:00",
  "updated_at": "2026-09-15T12:00:00+00:00"
}
```

Create/update body: `{ title, category, status, date, start_time, end_time,
sub_tasks: [{ title, is_done }] }` (sub-tasks get server-generated ids).

### Notes

| Method | Endpoint            | Body                 |
| ------ | ------------------- | -------------------- |
| GET    | `/api/notes`        | —                    |
| POST   | `/api/notes`        | `title, content`     |
| GET    | `/api/notes/{note}` | —                    |
| PUT    | `/api/notes/{note}` | `title?`, `content?` |
| DELETE | `/api/notes/{note}` | —                    |

### Alerts (read-only — derived from tasks)

| Method | Endpoint               | Purpose                    |
| ------ | ---------------------- | -------------------------- |
| GET    | `/api/alerts`          | list alerts, newest first  |
| POST   | `/api/alerts/read-all` | mark all as read           |

`GET /api/alerts` reconciles before it lists. That matters: a deadline can pass
while the app is open without anything being written, so if the sweep only ran
after task mutations, an "overdue" warning would not appear until the user next
edited a task. Reconciliation is idempotent and leaves existing `is_read` flags
alone, so sweeping on read costs one query and can neither duplicate nor
resurrect an alert. (See `AlertController::index`.)

---

## Mapping to the Flutter app

**This is done.** The Flutter app in [`../manager`](../manager) is wired to this
API; [`../manager/API_INTEGRATION.md`](../manager/API_INTEGRATION.md) is the
client-side reference for the contract.

The backend is snake_case (idiomatic Laravel) while the Flutter app uses
camelCase, so the app's models carry two mappers: `fromApi`/`toApi` for the wire
format below, and `fromJson`/`toJson` for its own local cache.

| Flutter app (camelCase) | Backend (snake_case)          | Notes                            |
| ----------------------- | ----------------------------- | -------------------------------- |
| `startTime`             | `start_time`                  | `"HH:MM"` string, both sides      |
| `endTime`               | `end_time`                    | `"HH:MM"` string, both sides      |
| `subTasks`              | `sub_tasks`                   | array of `{title, is_done}`       |
| `isDone`                | `is_done`                     |                                   |
| `isSuccess` / `isRead`  | `is_success` / `is_read`      |                                   |
| `createdAt` / `updatedAt` | `created_at` / `updated_at` | ISO-8601 UTC, converted to local  |
| `date` (`DateTime`)     | `date` (`"YYYY-MM-DD"`)       | `AppDateUtils.isoDate` on the way out |

Two deliberate divergences worth knowing:

1. **Sub-task ids** — the server generates them (UUIDs) and returns them; on
   update the server *replaces* the sub-task list, so ids change. The client
   therefore only ever toggles using ids from the most recent response, and never
   sends sub-task ids back.
2. **Ids are server-assigned** — `Task`, `SubTask`, `Note` and `Alert` ids are
   UUIDs generated here; `users.id` is an auto-increment integer. The client sends
   no ids on create.

---

## Password hashing: bcrypt vs PBKDF2

The Flutter app **no longer hashes passwords at all**. It used to store
PBKDF2-HMAC-SHA256 hashes (100,000 iterations, per-user salt) because it ran
fully offline with a local-only account store; now that it is wired to this API
it sends the plaintext password once, over the wire, and keeps only the opaque
Sanctum token that comes back. Verification and hashing are entirely
server-side.

This backend uses Laravel's default **bcrypt** via the `hashed` cast on `User`:

```php
// User model
protected function casts(): array
{
    return ['password' => 'hashed'];  // Hash::make() on set, automatically
}
```

So `User::create(['password' => $plain])` stores a bcrypt hash, and login
verifies with `Hash::check($plain, $user->password)`. bcrypt is the idiomatic
Laravel choice — slow to brute-force and battle-tested.

If you ever wanted a client that *did* hash locally, you can register a custom
hasher:

```php
// PBKDF2-HMAC-SHA256, if a client needed to derive keys on-device
$hash = base64_encode(hash_pbkdf2('sha256', $password, base64_decode($salt), 100000, 32, true));
```

Laravel's `Hash` manager is extensible (`Hash::extend(...)`), but for a normal
server app bcrypt (or Argon2) is the right default.

**HTTPS is the remaining gap.** The dev setup sends credentials in plaintext over
the LAN, which is fine on a trusted network with a debug build and unacceptable
anywhere else — see [Going to production](#going-to-production).

---

## Alerts: derived, not stored

Alerts are the most interesting piece. Rather than the client (or some job)
*creating* alert rows, the server **derives** them from task state and then
**reconciles** the `alerts` table to match:

- A completed task → a "Nice work" congratulation (`is_success = true`).
- An unfinished task past its deadline (within a 7-day grace window) → an
  "overdue" warning.
- An unfinished task due within an hour → a "time is running out" warning.

`app/Services/TaskAlertsService.php` does the derivation; `TaskController` calls
`syncForUser()` after every task mutation (create/update/delete/toggle).

The clever part is **reconciliation** (see `syncForUser`): each desired alert has
a deterministic `key` like `task-completed-<uuid>`. The service then:

1. **Deletes** any managed alert whose key is no longer desired (its cause went
   away — e.g. you finished the overdue task).
2. **Inserts** only genuinely new keys, leaving existing rows (and their
   `is_read` flag) untouched.

Because it reconciles rather than appends, re-running the scan can never
duplicate an alert, and a warning disappears on its own once its cause does —
exactly the behaviour the Flutter app implements in its `TaskAlertsService`.

---

## Testing

```bash
php artisan test
```

The suite covers the full API: registration/login/logout/reset, task CRUD,
sub-task auto-completion, cross-user isolation, note CRUD, and alert
reconciliation. It runs against in-memory SQLite (see `phpunit.xml`), so it's
fast and needs no MySQL.

A notable detail in `tests/Feature/AuthTest.php`: because Laravel's auth manager
is a singleton shared across requests *within a test*, the logout test flushes
the cached guard (`$this->app['auth']->forgetGuards()`) before verifying the
revoked token is rejected — mimicking a real request's fresh container.

---

## Going to production

This is a learning project. Before deploying it for real, consider:

- **nginx + PHP-FPM** instead of `php artisan serve` (which is a single-threaded
  dev server). A production image runs `php-fpm` behind an nginx reverse proxy.
- **Policies** for authorization — replace the inline `abort_unless(...)`
  ownership checks with `app/Policies/TaskPolicy.php` + `$this->authorize()`.
- **A real password reset** — the current `/reset-password` mirrors the app's
  offline convenience (no verification). Production would email a signed,
  single-use link (Laravel ships `PasswordReset` + `MustVerifyEmail` for this).
- **Rate limiting tuned per-route** rather than one global limiter.
- **Stable `APP_KEY`** — keep the key out of `docker-compose.yml` and inject it
  via a secret manager (Docker secrets / environment).
- **Observability** — queues for slow jobs, structured logging, and a health
  endpoint (Laravel already exposes `/up`).

---

## License

Private project — not published to Packagist.
