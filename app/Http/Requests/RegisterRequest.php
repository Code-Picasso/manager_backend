<?php

namespace App\Http\Requests;

use App\Rules\StrongPassword;
use Illuminate\Foundation\Http\FormRequest;

/** Validates the "create an account" request. */
class RegisterRequest extends FormRequest
{
    /** Anyone may register. */
    public function authorize(): bool
    {
        return true;
    }

    /** The validation rules for a new account. */
    public function rules(): array
    {
        return [
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'string', 'email', 'max:255', 'unique:users,email'],
            'password' => ['required', 'string', new StrongPassword],
        ];
    }
}
