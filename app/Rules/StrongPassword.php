<?php

namespace App\Rules;

use Closure;
use Illuminate\Contracts\Validation\ValidationRule;

/** A reusable validation rule enforcing the app's password policy. */
class StrongPassword implements ValidationRule
{
    private const SPECIAL_CHARACTERS = '/[~!@#\$%^&*()_+\-=\[\]{}|;:,.<>?]/';

    public function validate(string $attribute, mixed $value, Closure $fail): void
    {
        if (! is_string($value)) {
            $fail('The :attribute must be a string.');

            return;
        }

        if (strlen($value) < 6) {
            $fail('The :attribute must be at least 6 characters.');

            return;
        }

        if (! preg_match('/[A-Z]/', $value)) {
            $fail('The :attribute must contain at least one uppercase letter.');

            return;
        }

        if (! preg_match('/[0-9]/', $value)) {
            $fail('The :attribute must contain at least one number.');

            return;
        }

        if (! preg_match(self::SPECIAL_CHARACTERS, $value)) {
            $fail('The :attribute must contain at least one special character.');

            return;
        }
    }
}
