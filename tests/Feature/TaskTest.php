<?php

namespace Tests\Feature;

use App\Models\Task;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class TaskTest extends TestCase
{
    use RefreshDatabase;

    private User $user;

    protected function setUp(): void
    {
        parent::setUp();
        $this->user = User::factory()->create();
        $this->actingAs($this->user, 'sanctum');
    }

    /** @return array<string, mixed> */
    private function validTask(array $overrides = []): array
    {
        return array_merge([
            'title' => 'Write tests',
            'description' => 'Cover the API',
            'category' => 'work',
            'status' => 'today',
            'date' => '2026-09-15',
            'start_time' => '09:00',
            'end_time' => '10:00',
            'sub_tasks' => [
                ['title' => 'First', 'is_done' => false],
                ['title' => 'Second', 'is_done' => false],
            ],
        ], $overrides);
    }

    public function test_user_can_create_task(): void
    {
        $response = $this->postJson('/api/tasks', $this->validTask());

        $response->assertStatus(201)
            ->assertJsonPath('title', 'Write tests')
            ->assertJsonCount(2, 'sub_tasks');

        $this->assertDatabaseHas('tasks', ['title' => 'Write tests']);
        $this->assertDatabaseCount('sub_tasks', 2);
    }

    public function test_create_task_validates_category_and_status(): void
    {
        $this->postJson('/api/tasks', $this->validTask(['category' => 'nope']))
            ->assertStatus(422)->assertJsonValidationErrors('category');

        $this->postJson('/api/tasks', $this->validTask(['status' => 'nope']))
            ->assertStatus(422)->assertJsonValidationErrors('status');
    }

    public function test_user_can_list_own_tasks(): void
    {
        Task::factory()->count(3)->create(['user_id' => $this->user->id]);

        $this->getJson('/api/tasks')->assertOk()->assertJsonCount(3);
    }

    public function test_user_cannot_see_another_users_task(): void
    {
        $other = User::factory()->create();
        $task = Task::factory()->create(['user_id' => $other->id]);

        $this->getJson("/api/tasks/{$task->id}")->assertStatus(404);
    }

    public function test_user_can_update_a_task(): void
    {
        $task = Task::factory()->create(['user_id' => $this->user->id]);

        $this->putJson("/api/tasks/{$task->id}", $this->validTask(['title' => 'Updated']))
            ->assertOk()
            ->assertJsonPath('title', 'Updated');
    }

    public function test_a_task_can_be_created_without_a_description(): void
    {
        // `tasks.description` is NOT NULL, so this must store an empty string.
        $payload = $this->validTask();
        unset($payload['description']);

        $this->postJson('/api/tasks', $payload)
            ->assertCreated()
            ->assertJsonPath('description', '');

        $this->assertDatabaseHas('tasks', ['description' => '']);
    }

    public function test_a_task_description_can_be_cleared(): void
    {
        $task = Task::factory()->create([
            'user_id' => $this->user->id,
            'description' => 'Something',
        ]);

        $this->putJson("/api/tasks/{$task->id}", $this->validTask(['description' => '']))
            ->assertOk()
            ->assertJsonPath('description', '');

        $this->assertDatabaseHas('tasks', ['id' => $task->id, 'description' => '']);
    }

    public function test_an_explicit_null_description_is_stored_as_empty(): void
    {
        // An explicit null description is still stored as an empty string.
        $this->postJson('/api/tasks', $this->validTask(['description' => null]))
            ->assertCreated()
            ->assertJsonPath('description', '');

        $this->assertDatabaseHas('tasks', ['description' => '']);
    }

    public function test_a_task_can_be_saved_with_no_sub_tasks(): void
    {
        // An empty sub_tasks array is accepted.
        $this->postJson('/api/tasks', $this->validTask(['sub_tasks' => []]))
            ->assertCreated()
            ->assertJsonPath('sub_tasks', []);
    }

    public function test_deleting_a_task_soft_deletes_it(): void
    {
        $task = Task::factory()->create(['user_id' => $this->user->id]);
        $task->subTasks()->create(['title' => 'A', 'position' => 0]);

        $this->deleteJson("/api/tasks/{$task->id}")->assertNoContent();

        // Soft-deleted: hidden from the list, with its sub-tasks intact.
        $this->assertSoftDeleted('tasks', ['id' => $task->id]);
        $this->assertDatabaseCount('sub_tasks', 1);
        $this->getJson('/api/tasks')->assertOk()->assertJsonCount(0);
    }

    public function test_user_can_restore_a_deleted_task(): void
    {
        $task = Task::factory()->create(['user_id' => $this->user->id]);
        $task->subTasks()->create(['title' => 'A', 'position' => 0]);

        $this->deleteJson("/api/tasks/{$task->id}")->assertNoContent();

        $this->postJson("/api/tasks/{$task->id}/restore")
            ->assertOk()
            ->assertJsonPath('id', $task->id)
            ->assertJsonCount(1, 'sub_tasks');

        $this->assertNotSoftDeleted('tasks', ['id' => $task->id]);
        $this->getJson('/api/tasks')->assertOk()->assertJsonCount(1);
    }

    public function test_user_cannot_restore_another_users_task(): void
    {
        $other = User::factory()->create();
        $task = Task::factory()->create(['user_id' => $other->id]);
        $task->delete();

        $this->postJson("/api/tasks/{$task->id}/restore")->assertStatus(404);
    }

    public function test_ticking_last_subtask_completes_the_task(): void
    {
        $task = Task::factory()->create(['user_id' => $this->user->id, 'status' => 'today']);
        $a = $task->subTasks()->create(['title' => 'A', 'is_done' => false, 'position' => 0]);
        $b = $task->subTasks()->create(['title' => 'B', 'is_done' => false, 'position' => 1]);

        $this->postJson("/api/tasks/{$task->id}/subtasks/{$a->id}/toggle")->assertOk();

        // Ticking the last remaining sub-task auto-completes the task.
        $this->postJson("/api/tasks/{$task->id}/subtasks/{$b->id}/toggle")
            ->assertOk()
            ->assertJsonPath('status', 'completed');
    }

    public function test_unticking_sends_completed_task_back_to_today(): void
    {
        $task = Task::factory()->create([
            'user_id' => $this->user->id,
            'status' => 'completed',
            'date' => now()->toDateString(),
        ]);
        $a = $task->subTasks()->create(['title' => 'A', 'is_done' => true, 'position' => 0]);

        $this->postJson("/api/tasks/{$task->id}/subtasks/{$a->id}/toggle")
            ->assertOk()
            ->assertJsonPath('status', 'today');
    }

    public function test_toggling_another_users_subtask_fails(): void
    {
        $other = User::factory()->create();
        $task = Task::factory()->create(['user_id' => $other->id]);
        $sub = $task->subTasks()->create(['title' => 'A', 'position' => 0]);

        $this->postJson("/api/tasks/{$task->id}/subtasks/{$sub->id}/toggle")
            ->assertStatus(404);
    }

    public function test_tasks_can_be_filtered_by_status(): void
    {
        Task::factory()->create(['user_id' => $this->user->id, 'status' => 'today']);
        Task::factory()->create(['user_id' => $this->user->id, 'status' => 'completed']);
        Task::factory()->create(['user_id' => $this->user->id, 'status' => 'comingUp']);

        $this->getJson('/api/tasks')->assertOk()->assertJsonCount(3);
        $this->getJson('/api/tasks?status=completed')->assertOk()->assertJsonCount(1);
        $this->getJson('/api/tasks?status=comingUp')->assertOk()->assertJsonCount(1);

        // An unknown status is rejected rather than silently returning nothing.
        $this->getJson('/api/tasks?status=nope')
            ->assertStatus(422)
            ->assertJsonValidationErrors('status');
    }
}
