<?php

namespace Database\Factories;

use App\Enums\TaskCategory;
use App\Enums\TaskStatus;
use App\Models\Task;
use App\Models\User;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Task>
 */
class TaskFactory extends Factory
{
    /**
     * Define the model's default state.
     *
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        return [
            'user_id' => User::factory(),
            'title' => fake()->sentence(3),
            'description' => fake()->sentence(),
            'category' => TaskCategory::Work,
            'status' => TaskStatus::Today,
            'date' => now()->toDateString(),
            'start_time' => '09:00',
            'end_time' => '10:00',
        ];
    }
}
