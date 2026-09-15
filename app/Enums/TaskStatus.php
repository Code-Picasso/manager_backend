<?php

namespace App\Enums;

/**
 * Where a task currently sits: on the Today board, the Coming up board, or the
 * Completed board.
 *
 * As with TaskCategory, the values match the Flutter app's `TaskStatus` enum
 * names exactly ('today', 'comingUp', 'completed'). Note that PHP enum case
 * names must be valid identifiers, so the case is `ComingUp` while its backed
 * value — the string actually stored and serialised — is the camelCase
 * 'comingUp' the frontend expects.
 */
enum TaskStatus: string
{
    case Today = 'today';
    case ComingUp = 'comingUp';
    case Completed = 'completed';

    public function label(): string
    {
        return match ($this) {
            self::Today => 'Today',
            self::ComingUp => 'Coming up',
            self::Completed => 'Completed',
        };
    }
}
