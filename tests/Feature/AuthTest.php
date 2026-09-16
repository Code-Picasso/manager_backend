<?php

namespace Tests\Feature;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AuthTest extends TestCase
{
    use RefreshDatabase;

    /** @var array<string, string> */
    private array $validUser = [
        'name' => 'Ada Lovelace',
        'email' => 'ada@example.com',
        'password' => 'Secret123!',
    ];

    public function test_user_can_register(): void
    {
        $response = $this->postJson('/api/register', $this->validUser);

        $response->assertStatus(201)
            ->assertJsonStructure(['user' => ['id', 'name', 'email'], 'token']);

        $this->assertDatabaseHas('users', ['email' => 'ada@example.com']);
    }

    public function test_register_rejects_weak_password(): void
    {
        $this->postJson('/api/register', [
            'name' => 'Ada',
            'email' => 'ada@example.com',
            'password' => 'weak',
        ])->assertStatus(422)->assertJsonValidationErrors('password');
    }

    public function test_register_rejects_duplicate_email(): void
    {
        User::factory()->create(['email' => 'ada@example.com']);

        $this->postJson('/api/register', $this->validUser)
            ->assertStatus(422)
            ->assertJsonValidationErrors('email');
    }

    public function test_email_is_normalised_to_lowercase(): void
    {
        $this->postJson('/api/register', [
            'name' => 'Ada',
            'email' => 'ADA@Example.COM',
            'password' => 'Secret123!',
        ])->assertCreated();

        $this->assertDatabaseHas('users', ['email' => 'ada@example.com']);
    }

    public function test_user_can_login(): void
    {
        $this->postJson('/api/register', $this->validUser);

        $this->postJson('/api/login', [
            'email' => 'ada@example.com',
            'password' => 'Secret123!',
        ])->assertOk()->assertJsonStructure(['user', 'token']);
    }

    public function test_login_with_wrong_password_fails(): void
    {
        $this->postJson('/api/register', $this->validUser);

        $this->postJson('/api/login', [
            'email' => 'ada@example.com',
            'password' => 'WrongPass1!',
        ])->assertStatus(401);
    }

    public function test_guests_are_rejected(): void
    {
        $this->getJson('/api/user')->assertStatus(401);
        $this->getJson('/api/tasks')->assertStatus(401);
    }

    public function test_authenticated_user_can_fetch_profile(): void
    {
        $user = User::factory()->create();

        $this->actingAs($user, 'sanctum')
            ->getJson('/api/user')
            ->assertOk()
            ->assertJsonPath('email', $user->email);
    }

    public function test_user_can_update_profile(): void
    {
        $user = User::factory()->create();

        $this->actingAs($user, 'sanctum')
            ->putJson('/api/user', ['name' => 'New Name'])
            ->assertOk()
            ->assertJsonPath('name', 'New Name');
    }

    public function test_user_can_logout_and_token_stops_working(): void
    {
        $user = User::factory()->create();
        $token = $user->createToken('access-token')->plainTextToken;

        $this->withToken($token)->postJson('/api/logout')->assertOk();

        // Flush the cached guard so the next request re-checks the deleted token.
        $this->app['auth']->forgetGuards();

        // The revoked token must no longer authenticate.
        $this->withToken($token)->getJson('/api/user')->assertStatus(401);
    }

    public function test_user_can_reset_password(): void
    {
        $this->postJson('/api/register', $this->validUser);

        $this->postJson('/api/reset-password', [
            'email' => 'ada@example.com',
            'password' => 'NewSecret123!',
        ])->assertOk();

        $this->postJson('/api/login', [
            'email' => 'ada@example.com',
            'password' => 'NewSecret123!',
        ])->assertOk();
    }

    public function test_user_can_deactivate_account(): void
    {
        $user = User::factory()->create();

        $this->actingAs($user, 'sanctum')->deleteJson('/api/user')->assertOk();

        $this->assertDatabaseMissing('users', ['id' => $user->id]);
    }

    public function test_user_can_change_password(): void
    {
        // UserFactory's default password is "password".
        $user = User::factory()->create();

        $this->actingAs($user, 'sanctum')->postJson('/api/change-password', [
            'current_password' => 'password',
            'password' => 'NewSecret123!',
        ])->assertOk();

        $this->postJson('/api/login', [
            'email' => $user->email,
            'password' => 'NewSecret123!',
        ])->assertOk();
    }

    public function test_change_password_requires_the_current_password(): void
    {
        $user = User::factory()->create();

        $this->actingAs($user, 'sanctum')->postJson('/api/change-password', [
            'current_password' => 'not-my-password',
            'password' => 'NewSecret123!',
        ])->assertStatus(422)->assertJsonValidationErrors('current_password');
    }

    public function test_change_password_enforces_strength(): void
    {
        $user = User::factory()->create();

        $this->actingAs($user, 'sanctum')->postJson('/api/change-password', [
            'current_password' => 'password',
            'password' => 'weak',
        ])->assertStatus(422)->assertJsonValidationErrors('password');
    }

    public function test_change_password_revokes_other_sessions_only(): void
    {
        $user = User::factory()->create();
        $current = $user->createToken('current')->plainTextToken;
        $user->createToken('other');

        $this->withToken($current)->postJson('/api/change-password', [
            'current_password' => 'password',
            'password' => 'NewSecret123!',
        ])->assertOk();

        // The other device's token is gone...
        $this->assertDatabaseMissing('personal_access_tokens', ['name' => 'other']);

        // ...but the session that made the change stays signed in.
        $this->app['auth']->forgetGuards();
        $this->withToken($current)->getJson('/api/user')->assertOk();
    }

    public function test_guests_cannot_change_password(): void
    {
        $this->postJson('/api/change-password', [
            'current_password' => 'password',
            'password' => 'NewSecret123!',
        ])->assertStatus(401);
    }
}
