<?php

namespace App\Http\Requests;

use App\Enums\TaskCategory;
use App\Enums\TaskStatus;
use App\Http\Requests\Concerns\NormalizesEmptyText;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

/** Validates the "update a task" request. */
class UpdateTaskRequest extends FormRequest
{
    use NormalizesEmptyText;

    public function authorize(): bool
    {
        return true;
    }

    /** Normalizes an empty description field to an empty string. */
    protected function prepareForValidation(): void
    {
        $this->normalizeEmptyText(['description']);
    }

    public function rules(): array
    {
        return [
            'title' => ['required', 'string', 'max:255'],
            'description' => ['sometimes', 'nullable', 'string'],
            'category' => ['required', Rule::enum(TaskCategory::class)],
            'status' => ['required', Rule::enum(TaskStatus::class)],
            'date' => ['required', 'date'],
            'start_time' => ['required', 'date_format:H:i'],
            'end_time' => ['required', 'date_format:H:i'],
            'sub_tasks' => ['sometimes', 'array'],
            'sub_tasks.*.title' => ['required', 'string', 'max:255'],
            'sub_tasks.*.is_done' => ['sometimes', 'boolean'],
        ];
    }
}
