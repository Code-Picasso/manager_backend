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

/** Handles authentication and profile management. */
class AuthController extends Controller
{
    /** Create an account and issue an access token. */
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

    /** Authenticate with email and password, then issue a token. */
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

    /** Return the authenticated user. */
    public function user(Request $request): UserResource
    {
        return new UserResource($request->user());
    }

    /** Update the authenticated user's display name. */
    public function updateProfile(UpdateProfileRequest $request): UserResource
    {
        $request->user()->update($request->validated());

        return new UserResource($request->user()->fresh());
    }

    /** Revoke the token used for the current request. */
    public function logout(Request $request): JsonResponse
    {
        $request->user()->currentAccessToken()->delete();

        return response()->json(['message' => 'Logged out.']);
    }

    /** Reset a user's password to the supplied value. */
    public function resetPassword(ResetPasswordRequest $request): JsonResponse
    {
        $user = User::where('email', strtolower(trim($request->email)))->firstOrFail();

        // The `hashed` cast re-hashes the new password before it is stored.
        $user->update(['password' => $request->password]);

        return response()->json(['message' => 'Password reset.']);
    }

    /** Change the authenticated user's password. */
    public function changePassword(ChangePasswordRequest $request): JsonResponse
    {
        $user = $request->user();

        if (! Hash::check($request->current_password, $user->password)) {
            // Returns a 422 on the current_password field.
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

    /** Permanently delete the account and revoke all its tokens. */
    public function deactivate(Request $request): JsonResponse
    {
        $request->user()->tokens()->delete();
        $request->user()->delete();

        return response()->json(['message' => 'Account deactivated.']);
    }
}
