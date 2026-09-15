<?php

namespace App\Http\Resources;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

/**
 * Shapes a User model into the API's JSON representation.
 *
 * A JsonResource is a transformation layer between your Eloquent models and the
 * JSON the client sees. It keeps database concerns (column names, hidden
 * fields) separate from API concerns (what shape the client expects).
 */
class UserResource extends JsonResource
{
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'id' => $this->id,
            'name' => $this->name,
            'email' => $this->email,
        ];
    }
}
