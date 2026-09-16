<?php

namespace App\Http\Controllers;

use App\Http\Resources\AlertResource;
use App\Services\TaskAlertsService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;

/** Read-only endpoints for alerts derived from tasks. */
class AlertController extends Controller
{
    public function __construct(private readonly TaskAlertsService $alerts) {}

    /** List the user's alerts, newest first. */
    public function index(Request $request): AnonymousResourceCollection
    {
        $user = $request->user();

        try {
            $this->alerts->syncForUser($user);
        } catch (\Throwable $e) {
            // Falls back to the stored alerts if the sync fails.
            report($e);
        }

        $alerts = $user
            ->alerts()
            ->latest('created_at')
            ->get();

        return AlertResource::collection($alerts);
    }

    /** Mark every alert as read. */
    public function markAllRead(Request $request): JsonResponse
    {
        $request->user()
            ->alerts()
            ->where('is_read', false)
            ->update(['is_read' => true]);

        return response()->json(['message' => 'All alerts marked as read.']);
    }
}
