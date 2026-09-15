<?php

namespace App\Http\Controllers;

use App\Enums\TaskStatus;
use App\Http\Requests\StoreTaskRequest;
use App\Http\Requests\UpdateTaskRequest;
use App\Http\Resources\TaskResource;
use App\Models\SubTask;
use App\Models\Task;
use App\Models\User;
use App\Services\TaskAlertsService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;
use Illuminate\Http\Response;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\Rule;

/**
 * CRUD for tasks, plus the sub-task toggle that drives status auto-derivation.
 *
 * Notice two patterns worth internalising:
 *   - Route model binding: type-hinting `Task $task` makes Laravel look the row
 *     up by the `{task}` route segment (throwing 404 if absent).
 *   - Ownership scoping: every method then checks the bound task belongs to the
 *     authenticated user, so one user can never read or modify another's rows.
 */
class TaskController extends Controller
{
    /**
     * Constructor injection: Laravel's service container resolves the
     * TaskAlertsService dependency and passes it in automatically.
     */
    public function __construct(private readonly TaskAlertsService $alerts) {}

    /**
     * List the authenticated user's tasks, oldest date first, with sub-tasks
     * eager-loaded so serialisation does not issue one query per task.
     *
     * Supports an optional `?status=today|comingUp|completed` filter so the
     * app's board tabs can be served server-side instead of filtering a full
     * list on the device.
     */
    public function index(Request $request): AnonymousResourceCollection
    {
        // Validating the query string means an unknown status returns a 422
        // rather than a silent empty list. (A heavier filter set would move to
        // a Form Request, like the write endpoints.)
        $validated = $request->validate([
            'status' => ['sometimes', Rule::enum(TaskStatus::class)],
        ]);

        $query = $request->user()
            ->tasks()
            ->with('subTasks')
            ->orderBy('date')
            ->orderBy('created_at');

        if (isset($validated['status'])) {
            $query->where('status', $validated['status']);
        }

        return TaskResource::collection($query->get());
    }

    /**
     * Create a task (and its sub-tasks) in a single database transaction.
     */
    public function store(StoreTaskRequest $request): JsonResponse
    {
        $data = $request->validated();
        $subTasks = $data['sub_tasks'] ?? [];
        unset($data['sub_tasks']);

        // A transaction makes "task + sub-tasks" atomic: if inserting a
        // sub-task fails, the task insert is rolled back too.
        $task = DB::transaction(function () use ($request, $data, $subTasks) {
            $task = $request->user()->tasks()->create($data);

            $this->replaceSubTasks($task, $subTasks);

            // Recompute the status from the sub-task state before persisting.
            $task->load('subTasks')->withAutoStatus()->save();

            return $task;
        });

        $this->syncAlerts($request->user());

        return (new TaskResource($task))->response()->setStatusCode(201);
    }

    /**
     * Show a single task.
     */
    public function show(Request $request, Task $task): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        return new TaskResource($task->load('subTasks'));
    }

    /**
     * Update a task. Mirrors the Flutter app's full-replacement semantics: the
     * submitted sub-task list replaces the existing one.
     */
    public function update(UpdateTaskRequest $request, Task $task): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        $data = $request->validated();
        $subTasks = $data['sub_tasks'] ?? [];
        unset($data['sub_tasks']);

        DB::transaction(function () use ($task, $data, $subTasks) {
            $task->update($data);
            $this->replaceSubTasks($task, $subTasks);
            $task->load('subTasks')->withAutoStatus()->save();
        });

        $this->syncAlerts($request->user());

        return new TaskResource($task->fresh('subTasks'));
    }

    /**
     * Delete a task.
     *
     * This is a *soft* delete: the row keeps its id and its sub-tasks, but is
     * hidden from every normal query. That is what makes the app's "Undo"
     * button possible — see restore() below.
     */
    public function destroy(Request $request, Task $task): Response
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        $task->delete();

        $this->syncAlerts($request->user());

        return response()->noContent();
    }

    /**
     * Restore a soft-deleted task (the app's "Undo" action).
     *
     * Route model binding reaches soft-deleted rows here because the route is
     * declared with ->withTrashed(). Because the sub-tasks were never deleted,
     * the task comes back exactly as it was.
     */
    public function restore(Request $request, Task $task): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        $task->restore();

        $this->syncAlerts($request->user());

        return new TaskResource($task->load('subTasks'));
    }

    /**
     * Flip one sub-task's done state, then re-derive the parent's status.
     *
     * This is the single place "tick a box" happens; because it funnels through
     * Task::withAutoStatus, ticking the last sub-task completes the task and
     * un-ticking one on a completed task sends it back to Today / Coming up.
     */
    public function toggleSubTask(Request $request, Task $task, SubTask $subTask): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);
        abort_unless($subTask->task_id === $task->id, 404);

        $subTask->update(['is_done' => ! $subTask->is_done]);

        $task->load('subTasks')->withAutoStatus()->save();

        $this->syncAlerts($request->user());

        return new TaskResource($task);
    }

    /**
     * Replace a task's sub-task list with the supplied array, preserving order
     * via the `position` column. The server is authoritative for sub-task ids,
     * so old rows are deleted and new rows are created with fresh UUIDs.
     */
    private function replaceSubTasks(Task $task, array $subTasks): void
    {
        $task->subTasks()->delete();

        foreach ($subTasks as $index => $subTask) {
            $task->subTasks()->create([
                'title' => $subTask['title'],
                'is_done' => $subTask['is_done'] ?? false,
                'position' => $index,
            ]);
        }
    }

    /**
     * Reconcile the user's alerts after a task change.
     *
     * Deliberately best-effort: if alert bookkeeping fails we log it rather
     * than failing the task write that triggered it (mirroring the Flutter
     * app's `_syncAlerts`).
     */
    private function syncAlerts(User $user): void
    {
        try {
            $this->alerts->syncForUser($user);
        } catch (\Throwable $e) {
            report($e);
        }
    }
}
