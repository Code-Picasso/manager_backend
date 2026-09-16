<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/** A free-form note owned by a user, with a client-side accent colour. */
class Note extends Model
{
    use HasFactory, HasUuids;

    /** The `content` column default. @var array<string, mixed> */
    protected $attributes = [
        'content' => '',
    ];

    /** @var list<string> */
    protected $fillable = [
        'user_id',
        'title',
        'content',
    ];

    /** True when the note has any non-whitespace body text. */
    public function hasContent(): bool
    {
        return trim((string) $this->content) !== '';
    }

    /** The user who owns this note. */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
