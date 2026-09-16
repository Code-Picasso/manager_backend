<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/** A notification shown to the user, derived from task state. */
class Alert extends Model
{
    use HasFactory, HasUuids;

    /** @var list<string> */
    protected $fillable = [
        'user_id',
        'key',
        'message',
        'is_success',
        'group',
        'is_read',
    ];

    /** @return array<string, string> */
    protected function casts(): array
    {
        return [
            'is_success' => 'boolean',
            'is_read' => 'boolean',
        ];
    }

    /** The user who owns this alert. */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
