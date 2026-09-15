<?php

namespace App\Models;

use App\Enums\TaskCategory;
use App\Enums\TaskStatus;
use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Support\Carbon;

/**
 * A task with an optional checklist of sub-tasks.
 *
 * The interesting logic here is the *derived* status: ticking the last
 * sub-task completes the task, and un-ticking one on a completed task sends it
 * back to Today or Coming up depending on its date. This is the same rule the
 * Flutter app implements in `Task.withAutoStatus`; it lives on the model so the
 * behaviour exists in exactly one place, no matter which endpoint mutates a
 * task.
 */
class Task extends Model
{
    use HasFactory, HasUuids, SoftDeletes;

    /**
     * @var list<string>
     */
    protected $fillable = [
        'user_id',
        'title',
        'description',
        'category',
        'status',
        'date',
        'start_time',
        'end_time',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            // Backed enums: stored as their string value, read back as objects.
            'category' => TaskCategory::class,
            'status' => TaskStatus::class,
            // 'date' -> a Carbon instance (Laravel's DateTime wrapper).
            'date' => 'date',
        ];
    }

    /**
     * The user who owns this task.
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    /**
     * The task's sub-tasks, ordered by their stored position.
     *
     * The camelCase name mirrors the frontend's `subTasks` JSON field; Eloquent
     * resolves `$task->subTasks` through this method.
     */
    public function subTasks(): HasMany
    {
        return $this->hasMany(SubTask::class)->orderBy('position');
    }

    // ---------------------------------------------------------------------
    // Derived state — mirrors the getters on the Flutter Task model.
    // ---------------------------------------------------------------------

    /**
     * Total number of sub-tasks, regardless of their done state.
     */
    public function totalSubTasks(): int
    {
        return $this->subTasks->count();
    }

    /**
     * Number of sub-tasks currently ticked off.
     */
    public function doneSubTasks(): int
    {
        return $this->subTasks->where('is_done', true)->count();
    }

    /**
     * True when the task has been moved into the Completed board.
     */
    public function isComplete(): bool
    {
        return $this->status === TaskStatus::Completed;
    }

    /**
     * True when the task has sub-tasks and every one of them is ticked.
     */
    public function allSubTasksDone(): bool
    {
        return $this->subTasks->isNotEmpty()
            && $this->subTasks->every(fn (SubTask $subTask) => $subTask->is_done);
    }

    /**
     * Completion progress as a value between 0.0 and 1.0.
     *
     * A task with no sub-tasks is either fully done (1) or not started (0);
     * otherwise it is the ratio of done sub-tasks to total.
     *
     * Written as an Eloquent accessor so it reads like a plain attribute:
     * `$task->progress`. (Accessors are the idiomatic way to expose a computed
     * value that has no backing column.)
     */
    public function getProgressAttribute(): float
    {
        $total = $this->totalSubTasks();

        if ($total === 0) {
            return $this->isComplete() ? 1.0 : 0.0;
        }

        return $this->doneSubTasks() / $total;
    }

    /**
     * The moment this task is due — its date at its end time.
     *
     * A malformed end time falls back to the end of the day rather than
     * throwing, so one bad record can't break the alert scan (the same
     * defensive behaviour the Flutter model has).
     */
    public function getDeadlineAttribute(): Carbon
    {
        $parts = array_pad(explode(':', (string) $this->end_time), 2, null);

        $hour = is_numeric($parts[0]) ? (int) $parts[0] : 23;
        $minute = is_numeric($parts[1]) ? (int) $parts[1] : 59;

        return $this->date->copy()->setTime($hour, $minute, 0);
    }

    /**
     * Re-derives the task's status from its sub-task progress.
     *
     * This is a direct port of `Task.withAutoStatus` from the Flutter app:
     *  - ticking the last sub-task completes the task;
     *  - un-ticking one on a completed task sends it back to Today or
     *    Coming up depending on whether its date is in the future;
     *  - tasks without sub-tasks are left untouched.
     *
     * Unlike the Flutter version (which returns a new immutable object), an
     * Eloquent model is mutable, so this sets `$this->status` in place and
     * returns `$this` for chaining. Call it before persisting the model.
     */
    public function withAutoStatus(?Carbon $now = null): static
    {
        if ($this->subTasks->isEmpty()) {
            return $this;
        }

        if ($this->allSubTasksDone()) {
            if (! $this->isComplete()) {
                $this->status = TaskStatus::Completed;
            }

            return $this;
        }

        if (! $this->isComplete()) {
            return $this;
        }

        $reference = $now ?? Carbon::now();
        $today = $reference->copy()->startOfDay();
        $due = $this->date->copy()->startOfDay();

        $this->status = $due->isAfter($today)
            ? TaskStatus::ComingUp
            : TaskStatus::Today;

        return $this;
    }
}
