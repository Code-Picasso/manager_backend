<?php

namespace App\Enums;

/** The two buckets a task can belong to: Work or Personal. */
enum TaskCategory: string
{
    case Work = 'work';
    case Personal = 'personal';

    /** Human-readable label for this category. */
    public function label(): string
    {
        return match ($this) {
            self::Work => 'Work',
            self::Personal => 'Personal',
        };
    }
}
