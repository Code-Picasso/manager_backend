<?php

namespace App\Http\Requests;

use App\Http\Requests\Concerns\NormalizesEmptyText;
use Illuminate\Foundation\Http\FormRequest;

/** Validates the "update a note" request. */
class UpdateNoteRequest extends FormRequest
{
    use NormalizesEmptyText;

    public function authorize(): bool
    {
        return true;
    }

    /** Normalizes an empty content field to an empty string. */
    protected function prepareForValidation(): void
    {
        $this->normalizeEmptyText(['content']);
    }

    public function rules(): array
    {
        return [
            'title' => ['sometimes', 'required', 'string', 'max:255'],
            'content' => ['sometimes', 'nullable', 'string'],
        ];
    }
}
