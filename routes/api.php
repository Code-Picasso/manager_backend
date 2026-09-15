<?php

use App\Http\Controllers\AlertController;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\NoteController;
use App\Http\Controllers\TaskController;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| API Routes
|--------------------------------------------------------------------------
|
| Routes in this file are registered with the "api" middleware group and the
| "/api" URI prefix automatically (see bootstrap/app.php -> withRouting()).
|
| Auth model: Laravel Sanctum personal-access tokens. A client registers or
| logs in, receives a `token`, then sends it on every protected request as
| `Authorization: Bearer <token>`. The auth:sanctum middleware resolves the
| user from that token.
|
*/

// -------------------------------------------------------------------------
// Public routes — no token required.
// -------------------------------------------------------------------------

Route::post('/register', [AuthController::class, 'register']);
Route::post('/login', [AuthController::class, 'login']);
Route::post('/reset-password', [AuthController::class, 'resetPassword']);

// -------------------------------------------------------------------------
// Protected routes — require a valid Bearer token.
// -------------------------------------------------------------------------

Route::middleware('auth:sanctum')->group(function () {
    // Current user.
    Route::get('/user', [AuthController::class, 'user']);
    Route::put('/user', [AuthController::class, 'updateProfile']);
    Route::delete('/user', [AuthController::class, 'deactivate']);
    Route::post('/logout', [AuthController::class, 'logout']);
    Route::post('/change-password', [AuthController::class, 'changePassword']);

    // Tasks. apiResource() expands to index/show/store/update/destroy.
    Route::apiResource('tasks', TaskController::class);

    // A custom action on top of the resource: flip one sub-task's done state.
    Route::post('/tasks/{task}/subtasks/{subTask}/toggle', [TaskController::class, 'toggleSubTask']);

    // Undo a deletion. withTrashed() lets the route bind a soft-deleted task,
    // which is otherwise invisible to the model's default query scope.
    Route::post('/tasks/{task}/restore', [TaskController::class, 'restore'])->withTrashed();

    // Notes.
    Route::apiResource('notes', NoteController::class);

    // Alerts — read-only (they are derived from tasks).
    Route::get('/alerts', [AlertController::class, 'index']);
    Route::post('/alerts/read-all', [AlertController::class, 'markAllRead']);
});
