<?php

use App\Http\Controllers\AlertController;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\NoteController;
use App\Http\Controllers\TaskController;
use Illuminate\Support\Facades\Route;

// API routes — group and "/api" prefix come from bootstrap/app.php.

// Public routes — no token required.

Route::post('/register', [AuthController::class, 'register']);
Route::post('/login', [AuthController::class, 'login']);
Route::post('/reset-password', [AuthController::class, 'resetPassword']);

// Protected routes — require a valid Bearer token.

Route::middleware('auth:sanctum')->group(function () {
    // Current user.
    Route::get('/user', [AuthController::class, 'user']);
    Route::put('/user', [AuthController::class, 'updateProfile']);
    Route::delete('/user', [AuthController::class, 'deactivate']);
    Route::post('/logout', [AuthController::class, 'logout']);
    Route::post('/change-password', [AuthController::class, 'changePassword']);

    // Tasks — apiResource() expands to index/show/store/update/destroy.
    Route::apiResource('tasks', TaskController::class);

    // Flips one sub-task's done state.
    Route::post('/tasks/{task}/subtasks/{subTask}/toggle', [TaskController::class, 'toggleSubTask']);

    // Undo a deletion — withTrashed() lets the route bind a soft-deleted task.
    Route::post('/tasks/{task}/restore', [TaskController::class, 'restore'])->withTrashed();

    // Notes.
    Route::apiResource('notes', NoteController::class);

    // Alerts — read-only, derived from tasks.
    Route::get('/alerts', [AlertController::class, 'index']);
    Route::post('/alerts/read-all', [AlertController::class, 'markAllRead']);
});
