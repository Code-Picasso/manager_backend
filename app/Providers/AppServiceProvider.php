<?php

namespace App\Providers;

use Illuminate\Cache\RateLimiting\Limit;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;
use Illuminate\Support\Facades\RateLimiter;
use Illuminate\Support\ServiceProvider;

/**
 * The application's service provider.
 *
 * A service provider is Laravel's bootstrap hook: `register()` runs early for
 * container bindings, and `boot()` runs once everything is registered and is
 * the right place for rate-limit definitions, view composers, etc.
 */
class AppServiceProvider extends ServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        //
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        // By default Laravel wraps every JSON resource in a top-level "data"
        // key. The Flutter app expects plain objects/arrays, so we disable the
        // wrapper globally for a cleaner, directly-usable API shape.
        JsonResource::withoutWrapping();

        // Define the "api" rate limiter that the throttleApi() middleware (see
        // bootstrap/app.php) applies to every /api request. Each authenticated
        // user (or, for unauthenticated requests, each IP) gets 60 requests
        // per minute.
        RateLimiter::for('api', function (Request $request) {
            return Limit::perMinute(60)->by($request->user()?->id ?: $request->ip());
        });
    }
}
