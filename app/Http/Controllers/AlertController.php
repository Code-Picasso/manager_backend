<?php

namespace App\Http\Controllers;

use App\Http\Resources\AlertResource;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;

/**
 * Read endpoints for alerts.
 *
 * Alerts are derived from tasks and reconciled after every task mutation (see
 * TaskController + TaskAlertsService), so this controller only *reads* them —
 * there is no "create alert" endpoint.
 */
class AlertController extends Controller
{
    /**
     * List the user's alerts, newest first.
     */
    public function index(Request $request): AnonymousResourceCollection
    {
        $alerts = $request->user()
            ->alerts()
            ->latest('created_at')
            ->get();

        return AlertResource::collection($alerts);
    }

    /**
     * Mark every alert as read. Called when the alerts screen opens so the
     * unread badge clears.
     */
    public function markAllRead(Request $request): JsonResponse
    {
        $request->user()
            ->alerts()
            ->where('is_read', false)
            ->update(['is_read' => true]);

        return response()->json(['message' => 'All alerts marked as read.']);
    }
}
