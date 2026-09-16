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

/** A task with an optional checklist of sub-tasks. */
class Task extends Model
{
    use HasFactory, HasUuids, SoftDeletes;

    /** Column defaults a new task starts with. @var array<string, mixed> */
    protected $attributes = [
        'description' => '',
        'category' => 'work',
        'status' => 'today',
        'start_time' => '09:00',
        'end_time' => '10:00',
    ];

    /** @var list<string> */
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

    /** @return array<string, string> */
    protected function casts(): array
    {
        return [
            // Backed enums: stored as their string value, read back as objects.
            'category' => TaskCategory::class,
            'status' => TaskStatus::class,
            // 'date' -> a Carbon instance.
            'date' => 'date',
        ];
    }

    /** The user who owns this task. */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    /** The task's sub-tasks, ordered by their stored position. */
    public function subTasks(): HasMany
    {
        return $this->hasMany(SubTask::class)->orderBy('position');
    }

    // Derived state

    /** Total number of sub-tasks. */
    public function totalSubTasks(): int
    {
        return $this->subTasks->count();
    }

    /** Number of sub-tasks currently ticked off. */
    public function doneSubTasks(): int
    {
        return $this->subTasks->where('is_done', true)->count();
    }

    /** True when the task has been moved into the Completed board. */
    public function isComplete(): bool
    {
        return $this->status === TaskStatus::Completed;
    }

    /** True when the task has sub-tasks and every one of them is ticked. */
    public function allSubTasksDone(): bool
    {
        return $this->subTasks->isNotEmpty()
            && $this->subTasks->every(fn (SubTask $subTask) => $subTask->is_done);
    }

    /** Completion progress as a value between 0.0 and 1.0. */
    public function getProgressAttribute(): float
    {
        $total = $this->totalSubTasks();

        if ($total === 0) {
            return $this->isComplete() ? 1.0 : 0.0;
        }

        return $this->doneSubTasks() / $total;
    }

    /** The moment this task is due — its date at its end time. */
    public function getDeadlineAttribute(): Carbon
    {
        $parts = array_pad(explode(':', (string) $this->end_time), 2, null);

        $hour = is_numeric($parts[0]) ? (int) $parts[0] : 23;
        $minute = is_numeric($parts[1]) ? (int) $parts[1] : 59;

        return $this->date->copy()->setTime($hour, $minute, 0);
    }

    /** Re-derives the task's status from its sub-task progress. */
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
