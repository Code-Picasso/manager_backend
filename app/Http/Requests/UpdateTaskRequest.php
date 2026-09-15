<?php

namespace App\Http\Requests;

use App\Enums\TaskCategory;
use App\Enums\TaskStatus;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

/**
 * Validates the "update a task" request.
 *
 * The Flutter app's updateTask overwrites the whole task (including its
 * sub-task list), so this mirrors that full-replacement contract with the same
 * rules as creation. The controller replaces the sub-task list accordingly.
 */
class UpdateTaskRequest extends FormRequest
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
