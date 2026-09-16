<?php

namespace Tests\Feature;

use App\Models\Task;
use App\Models\User;
use App\Services\TaskAlertsService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AlertTest extends TestCase
{
    use RefreshDatabase;

    private User $user;

    private TaskAlertsService $service;

    protected function setUp(): void
    {
        parent::setUp();
        $this->user = User::factory()->create();
        $this->actingAs($this->user, 'sanctum');
        $this->service = app(TaskAlertsService::class);
    }

    public function test_completed_task_produces_a_congratulation(): void
    {
        $task = Task::factory()->create([
            'user_id' => $this->user->id,
            'status' => 'completed',
        ]);

        $this->service->syncForUser($this->user, now());

        $this->getJson('/api/alerts')
            ->assertOk()
            ->assertJsonCount(1)
            ->assertJsonPath('0.is_success', true)
            ->assertJsonPath('0.message', 'Nice work — you finished "'.$task->title.'".');
    }

    public function test_overdue_task_produces_a_warning(): void
    {
        Task::factory()->create([
            'user_id' => $this->user->id,
            'status' => 'today',
            'date' => now()->subDay()->toDateString(),
            'end_time' => '10:00',
        ]);

        $this->service->syncForUser($this->user, now());

        $this->getJson('/api/alerts')
            ->assertOk()
            ->assertJsonCount(1)
            ->assertJsonPath('0.is_success', false);
    }

    public function test_reconciliation_never_duplicates(): void
    {
        Task::factory()->create(['user_id' => $this->user->id, 'status' => 'completed']);

        // Run the scan twice — the result must still be a single alert.
        $this->service->syncForUser($this->user, now());
        $this->service->syncForUser($this->user, now());

        $this->assertDatabaseCount('alerts', 1);
    }

    public function test_reconciliation_removes_stale_alerts(): void
    {
        $task = Task::factory()->create(['user_id' => $this->user->id, 'status' => 'completed']);

        $this->service->syncForUser($this->user, now());
        $this->assertDatabaseCount('alerts', 1);

        // Delete the task — its congratulation must disappear on the next scan.
        $task->delete();
        $this->service->syncForUser($this->user, now());

        $this->assertDatabaseCount('alerts', 0);
    }

    public function test_mark_all_read(): void
    {
        Task::factory()->create(['user_id' => $this->user->id, 'status' => 'completed']);
        $this->service->syncForUser($this->user, now());

        $this->postJson('/api/alerts/read-all')->assertOk();

        $this->assertDatabaseHas('alerts', ['is_read' => true]);
    }

    public function test_listing_alerts_reconciles_without_a_task_change(): void
    {
        // No explicit sync: listing the alerts runs the scan.
        Task::factory()->create([
            'user_id' => $this->user->id,
            'status' => 'today',
            'date' => now()->subDay()->toDateString(),
            'end_time' => '10:00',
        ]);

        $this->assertDatabaseCount('alerts', 0);

        $this->getJson('/api/alerts')
            ->assertOk()
            ->assertJsonCount(1)
            ->assertJsonPath('0.is_success', false);
    }

    public function test_listing_alerts_never_resurrects_a_read_alert(): void
    {
        Task::factory()->create(['user_id' => $this->user->id, 'status' => 'completed']);

        // First read derives the alert, then the user dismisses it.
        $this->getJson('/api/alerts')->assertOk()->assertJsonCount(1);
        $this->postJson('/api/alerts/read-all')->assertOk();

        // Reading again re-runs the scan; the alert must stay read and singular.
        $this->getJson('/api/alerts')
            ->assertOk()
            ->assertJsonCount(1)
            ->assertJsonPath('0.is_read', true);

        $this->assertDatabaseCount('alerts', 1);
    }
}
