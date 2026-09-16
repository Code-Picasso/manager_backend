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

/** CRUD for tasks, plus the sub-task toggle that drives status auto-derivation. */
class TaskController extends Controller
{
    /** Injects the alert service. */
    public function __construct(private readonly TaskAlertsService $alerts) {}

    /** List the authenticated user's tasks, oldest first, optionally by status. */
    public function index(Request $request): AnonymousResourceCollection
    {
        // Validates the optional status query parameter.
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

    /** Create a task and its sub-tasks in a single transaction. */
    public function store(StoreTaskRequest $request): JsonResponse
    {
        $data = $request->validated();
        $subTasks = $data['sub_tasks'] ?? [];
        unset($data['sub_tasks']);

        // Task and sub-task inserts are atomic.
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

    /** Show a single task. */
    public function show(Request $request, Task $task): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        return new TaskResource($task->load('subTasks'));
    }

    /** Update a task, replacing its sub-task list with the submitted one. */
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

    /** Soft-delete a task, keeping its row and sub-tasks. */
    public function destroy(Request $request, Task $task): Response
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        $task->delete();

        $this->syncAlerts($request->user());

        return response()->noContent();
    }

    /** Restore a soft-deleted task. */
    public function restore(Request $request, Task $task): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);

        $task->restore();

        $this->syncAlerts($request->user());

        return new TaskResource($task->load('subTasks'));
    }

    /** Flip one sub-task's done state and re-derive the parent's status. */
    public function toggleSubTask(Request $request, Task $task, SubTask $subTask): TaskResource
    {
        abort_unless($task->user_id === $request->user()->id, 404);
        abort_unless($subTask->task_id === $task->id, 404);

        $subTask->update(['is_done' => ! $subTask->is_done]);

        $task->load('subTasks')->withAutoStatus()->save();

        $this->syncAlerts($request->user());

        return new TaskResource($task);
    }

    /** Replace a task's sub-tasks, preserving order via the position column. */
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

    /** Reconcile the user's alerts, logging failures instead of throwing. */
    private function syncAlerts(User $user): void
    {
        try {
            $this->alerts->syncForUser($user);
        } catch (\Throwable $e) {
            report($e);
        }
    }
}
