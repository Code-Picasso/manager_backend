<?php

namespace App\Http\Requests;

use App\Rules\StrongPassword;
use Illuminate\Foundation\Http\FormRequest;

/**
 * Validates the "change my password" request for a signed-in user.
 *
 * Unlike reset-password (which trusts a known email), this is the security-
 * correct flow: the caller must prove they know the *current* password before
 * a new one is accepted. The controller verifies `current_password`; if it is
 * wrong the request fails with a 422 on that field.
 */
class ChangePasswordRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'current_password' => ['required', 'string'],
            'password' => ['required', 'string', new StrongPassword],
        ];
    }
}
