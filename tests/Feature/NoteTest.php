<?php

namespace Tests\Feature;

use App\Models\Note;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class NoteTest extends TestCase
{
    use RefreshDatabase;

    private User $user;

    protected function setUp(): void
    {
        parent::setUp();
        $this->user = User::factory()->create();
        $this->actingAs($this->user, 'sanctum');
    }

    public function test_user_can_create_a_note(): void
    {
        $this->postJson('/api/notes', ['title' => 'Groceries', 'content' => 'Milk, eggs'])
            ->assertStatus(201)
            ->assertJsonPath('title', 'Groceries');

        $this->assertDatabaseHas('notes', ['title' => 'Groceries']);
    }

    public function test_user_can_list_own_notes(): void
    {
        Note::factory()->count(2)->create(['user_id' => $this->user->id]);

        $this->getJson('/api/notes')->assertOk()->assertJsonCount(2);
    }

    public function test_user_can_update_a_note(): void
    {
        $note = Note::factory()->create(['user_id' => $this->user->id]);

        $this->putJson("/api/notes/{$note->id}", ['content' => 'Updated body'])
            ->assertOk()
            ->assertJsonPath('content', 'Updated body');
    }

    public function test_user_can_delete_a_note(): void
    {
        $note = Note::factory()->create(['user_id' => $this->user->id]);

        $this->deleteJson("/api/notes/{$note->id}")->assertNoContent();

        $this->assertDatabaseMissing('notes', ['id' => $note->id]);
    }

    public function test_user_cannot_see_another_users_note(): void
    {
        $other = User::factory()->create();
        $note = Note::factory()->create(['user_id' => $other->id]);

        $this->getJson("/api/notes/{$note->id}")->assertStatus(404);
    }
}
