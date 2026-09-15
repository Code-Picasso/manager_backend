<?php

namespace App\Http\Requests;

use App\Rules\StrongPassword;
use Illuminate\Foundation\Http\FormRequest;

/**
 * Validates the "create an account" request.
 *
 * A Form Request centralises validation (and, via authorize(), authorization)
 * outside the controller. Laravel runs the rules *before* the controller
 * method executes and, on failure, returns a 422 JSON response automatically —
 * the controller can assume the data is already valid.
 */
class RegisterRequest extends FormRequest
{
    /**
     * Whether the current user is allowed to make this request. Registration is
     * public, so anyone may.
     */
    public function authorize(): bool
    {
        return true;
    }

    /**
     * The validation rules. The password policy lives in the reusable
     * StrongPassword rule object (shared with reset-password / change-password).
     */
    public function rules(): array
    {
        return [
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'string', 'email', 'max:255', 'unique:users,email'],
            'password' => ['required', 'string', new StrongPassword],
        ];
    }
}
