<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * A free-form note owned by a user, with a stable accent colour applied by the
 * client (so the colour is not persisted here — it is a presentation concern).
 *
 * `created_at` and `updated_at` are maintained automatically by Eloquent's
 * timestamp support: they are set on insert and update respectively, which
 * mirrors the Flutter Note model's `createdAt` / `updatedAt` fields.
 */
class Note extends Model
{
    use HasFactory, HasUuids;

    /**
     * @var list<string>
     */
    protected $fillable = [
        'user_id',
        'title',
        'content',
    ];

    /**
     * True when the note has any non-whitespace body text.
     *
     * Mirrors the `hasContent` getter on the Flutter Note model.
     */
    public function hasContent(): bool
    {
        return trim((string) $this->content) !== '';
    }

    /**
     * The user who owns this note.
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
