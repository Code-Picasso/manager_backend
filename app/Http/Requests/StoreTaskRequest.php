<?php

namespace App\Http\Requests;

use App\Enums\TaskCategory;
use App\Enums\TaskStatus;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

/**
 * Validates the "create a task" request.
 *
 * The field names are snake_case (idiomatic Laravel). The README includes a
 * table mapping them to the Flutter app's camelCase payload (start_time ->
 * startTime, sub_tasks -> subTasks, and so on).
 */
class StoreTaskRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'title' => ['required', 'string', 'max:255'],
            'description' => ['sometimes', 'nullable', 'string'],
            // Rule::enum rejects any string that isn't a valid TaskCategory
            // case value ('work' or 'personal').
            'category' => ['required', Rule::enum(TaskCategory::class)],
            'status' => ['required', Rule::enum(TaskStatus::class)],
            'date' => ['required', 'date'],
            // date_format:H:i enforces the "HH:MM" shape the app uses.
            'start_time' => ['required', 'date_format:H:i'],
            'end_time' => ['required', 'date_format:H:i'],
            'sub_tasks' => ['sometimes', 'array'],
            'sub_tasks.*.title' => ['required', 'string', 'max:255'],
            'sub_tasks.*.is_done' => ['sometimes', 'boolean'],
        ];
    }
}
