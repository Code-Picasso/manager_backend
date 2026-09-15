<?php

namespace App\Http\Controllers;

use App\Http\Requests\ChangePasswordRequest;
use App\Http\Requests\LoginRequest;
use App\Http\Requests\RegisterRequest;
use App\Http\Requests\ResetPasswordRequest;
use App\Http\Requests\UpdateProfileRequest;
use App\Http\Resources\UserResource;
use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;
use Illuminate\Validation\ValidationException;

/**
 * Handles authentication and profile management.
 *
 * A "controller" in Laravel is just a class whose methods handle HTTP requests.
 * Routing maps a URL + verb to one of these methods; the framework injects the
 * validated FormRequest (and any route-bound model) automatically.
 *
 * Token auth is provided by Laravel Sanctum: after registering or logging in,
 * the user receives a `plainTextToken` that the client sends on subsequent
 * requests as `Authorization: Bearer <token>`.
 */
class AuthController extends Controller
{
    /**
     * Create an account and log it in.
     *
     * The password is hashed automatically by the User model's `hashed` cast,
     * so we never touch the plaintext beyond validation.
     */
    public function register(RegisterRequest $request): JsonResponse
    {
        $data = $request->validated();
        $data['email'] = strtolower(trim($data['email']));

        $user = User::create($data);

        return response()->json([
            'user' => new UserResource($user),
            'token' => $user->createToken('access-token')->plainTextToken,
        ], 201);
    }

    /**
     * Authenticate with email + password and issue a token.
     *
     * We verify the password with Hash::check against the stored bcrypt hash,
     * and return a single, generic error either way — revealing whether an
     * email exists would let an attacker enumerate accounts. (The README
     * contrasts this with the Flutter app's friendlier per-case messages.)
     */
    public function login(LoginRequest $request): JsonResponse
    {
        $email = strtolower(trim($request->email));

        $user = User::where('email', $email)->first();

        if (! $user || ! Hash::check($request->password, $user->password)) {
            return response()->json([
                'message' => 'The provided credentials are incorrect.',
            ], 401);
        }

        return response()->json([
            'user' => new UserResource($user),
            'token' => $user->createToken('access-token')->plainTextToken,
        ]);
    }

    /**
     * Return the authenticated user (used by the app to restore a session).
     */
    public function user(Request $request): UserResource
    {
        return new UserResource($request->user());
    }

    /**
     * Update the authenticated user's display name.
     */
    public function updateProfile(UpdateProfileRequest $request): UserResource
    {
        $request->user()->update($request->validated());

        return new UserResource($request->user()->fresh());
    }

    /**
     * Revoke only the token used for the current request, leaving the account
     * (and any other active sessions) intact.
     */
    public function logout(Request $request): JsonResponse
    {
        $request->user()->currentAccessToken()->delete();

        return response()->json(['message' => 'Logged out.']);
    }

    /**
     * Reset a user's password to the supplied value.
     *
     * Mirrors the Flutter app's local-only convenience: there is no email
     * verification, so anyone who knows the email can reset it. The README
     * explains the production-grade flow this should become.
     */
    public function resetPassword(ResetPasswordRequest $request): JsonResponse
    {
        $user = User::where('email', strtolower(trim($request->email)))->firstOrFail();

        // The `hashed` cast re-hashes the new password before it is stored.
        $user->update(['password' => $request->password]);

        return response()->json(['message' => 'Password reset.']);
    }

    /**
     * Change the authenticated user's password.
     *
     * The caller must prove they know their *current* password — that is the
     * whole point of this endpoint (reset-password above only trusts an email).
     * On success every *other* token is revoked, so a stolen session cannot
     * outlive the password change; the token making this request stays valid so
     * the user isn't kicked out mid-session.
     */
    public function changePassword(ChangePasswordRequest $request): JsonResponse
    {
        $user = $request->user();

        if (! Hash::check($request->current_password, $user->password)) {
            // A ValidationException produces the same 422 shape as a failed
            // Form Request rule, but from inside the controller.
            throw ValidationException::withMessages([
                'current_password' => ['The current password is incorrect.'],
            ]);
        }

        $user->update(['password' => $request->password]);

        $currentTokenId = $user->currentAccessToken()?->id;

        $user->tokens()
            ->when($currentTokenId, fn ($query) => $query->where('id', '!=', $currentTokenId))
            ->delete();

        return response()->json(['message' => 'Password changed.']);
    }

    /**
     * Permanently delete the account, revoking all its tokens. The foreign-key
     * cascade removes the user's tasks, sub-tasks, notes and alerts.
     */
    public function deactivate(Request $request): JsonResponse
    {
        $request->user()->tokens()->delete();
        $request->user()->delete();

        return response()->json(['message' => 'Account deactivated.']);
    }
}
