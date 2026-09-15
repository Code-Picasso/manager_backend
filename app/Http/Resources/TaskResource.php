<?php

namespace App\Http\Resources;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

/**
 * Shapes a Task into JSON, re-nesting its sub-tasks the way the Flutter app
 * expects them (a `sub_tasks` array) even though they are stored in their own
 * table.
 */
class TaskResource extends JsonResource
{
    /**
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'id' => $this->id,
            'title' => $this->title,
            'description' => $this->description,
            // $this->category is a TaskCategory enum; ->value is the stored
            // string ('work' | 'personal') the app already understands.
            'category' => $this->category->value,
            'status' => $this->status->value,
            'date' => $this->date->toDateString(),
            'start_time' => $this->start_time,
            'end_time' => $this->end_time,
            // whenLoaded() only embeds the sub-tasks if the relation was
            // eager-loaded — otherwise it is omitted (and this also avoids
            // accidental N+1 queries during serialisation).
            'sub_tasks' => SubTaskResource::collection($this->whenLoaded('subTasks')),
            // The derived completion ratio (an Eloquent accessor on the model).
            'progress' => $this->progress,
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }
}
