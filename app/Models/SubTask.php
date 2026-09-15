<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * A single checklist item that lives inside a Task.
 *
 * In the Flutter app, sub-tasks are embedded as a JSON array on the task. A
 * relational database normalises them into their own table instead — each row
 * points at its parent via `task_id`. Laravel lets us keep the same JSON shape
 * the app expects by re-nesting them in the TaskResource, while enjoying real
 * foreign keys, cascade deletes and ordering here.
 */
class SubTask extends Model
{
    use HasFactory, HasUuids;

    /**
     * Columns that may be mass-assigned. Without listing a column here,
     * `SubTask::create([...])` would silently ignore it — a deliberate safety
     * valve against "mass assignment" attacks on user-supplied input.
     *
     * @var list<string>
     */
    protected $fillable = [
        'task_id',
        'title',
        'is_done',
        'position',
    ];

    /**
     * Attribute type casts: `is_done` is stored as 0/1 in the database but
     * exposed as a real PHP bool on the model.
     *
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'is_done' => 'boolean',
        ];
    }

    /**
     * The parent task this sub-task belongs to.
     */
    public function task(): BelongsTo
    {
        return $this->belongsTo(Task::class);
    }
}
