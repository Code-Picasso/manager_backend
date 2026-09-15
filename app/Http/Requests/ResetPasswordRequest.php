<?php

namespace App\Http\Requests;

use App\Rules\StrongPassword;
use Illuminate\Foundation\Http\FormRequest;

/**
 * Validates the "reset my password" request.
 *
 * In the Flutter app this is a local-only convenience (there is no email
 * verification, so anyone who knows the email can reset it). The API mirrors
 * that behaviour; the README notes how you would add a real email-verified
 * reset flow in production.
 */
class ResetPasswordRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'email' => ['required', 'string', 'email', 'exists:users,email'],
            // The new password gets the same strength rules as registration.
            'password' => ['required', 'string', new StrongPassword],
        ];
    }
}
