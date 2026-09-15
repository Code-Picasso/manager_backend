<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * A notification shown to the user.
 *
 * Alerts are *derived* from task state (a finished task earns a congratulation;
 * a task running out of time earns a warning) and reconciled against the task
 * list on every task change — exactly like the Flutter app does. See
 * App\Services\TaskAlertsService for the derivation and reconciliation logic.
 *
 * Two identifier concepts coexist here, which is worth understanding:
 *   - `id` is a generated UUID — the row's immutable, meaningless primary key.
 *   - `key` is a deterministic, human-readable string like
 *     "task-completed-<task-uuid>". It is what reconciliation uses to decide
 *     whether an alert already exists, so re-running the scan can never
 *     duplicate an alert.
 */
class Alert extends Model
{
    use HasFactory, HasUuids;

    /**
     * @var list<string>
     */
    protected $fillable = [
        'user_id',
        'key',
        'message',
        'is_success',
        'group',
        'is_read',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'is_success' => 'boolean',
            'is_read' => 'boolean',
        ];
    }

    /**
     * The user who owns this alert.
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
