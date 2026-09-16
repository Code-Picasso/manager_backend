<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/** A single checklist item that lives inside a task. */
class SubTask extends Model
{
    use HasFactory, HasUuids;

    /** Columns that may be mass-assigned. @var list<string> */
    protected $fillable = [
        'task_id',
        'title',
        'is_done',
        'position',
    ];

    /** Attribute type casts. @return array<string, string> */
    protected function casts(): array
    {
        return [
            'is_done' => 'boolean',
        ];
    }

    /** The parent task this sub-task belongs to. */
    public function task(): BelongsTo
    {
        return $this->belongsTo(Task::class);
    }
}
