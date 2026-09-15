<?php

namespace App\Enums;

/**
 * The two buckets a task can belong to: Work or Personal.
 *
 * This is a PHP "backed enum" — every case carries a string value. Because the
 * Task model casts its `category` column to this enum, Eloquent stores that
 * value in the database and, when reading a row back, hands you a
 * TaskCategory object instead of a raw string.
 *
 * The values ('work', 'personal') deliberately match the Flutter frontend's
 * `TaskCategory` enum names, so the JSON the API returns is byte-for-byte
 * compatible with what the app already expects.
 */
enum TaskCategory: string
{
    case Work = 'work';
    case Personal = 'personal';

    /**
     * Human-readable label used in UI-facing places (and available to the
     * frontend if it ever wants to drop its own copy of this mapping).
     */
    public function label(): string
    {
        return match ($this) {
            self::Work => 'Work',
            self::Personal => 'Personal',
        };
    }
}
