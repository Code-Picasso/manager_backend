<?php

namespace App\Enums;

/** Where a task currently sits: on the Today, Coming up or Completed board. */
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
