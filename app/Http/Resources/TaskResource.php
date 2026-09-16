<?php

namespace App\Http\Resources;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

/** Shapes a Task into JSON. */
class TaskResource extends JsonResource
{
    /** @return array<string, mixed> */
    public function toArray(Request $request): array
    {
        return [
            'id' => $this->id,
            'title' => $this->title,
            'description' => $this->description,
            // Serialised from the TaskCategory enum's stored string value.
            'category' => $this->category->value,
            'status' => $this->status->value,
            'date' => $this->date->toDateString(),
            'start_time' => $this->start_time,
            'end_time' => $this->end_time,
            // Sub-tasks appear only when the relation was eager-loaded.
            'sub_tasks' => SubTaskResource::collection($this->whenLoaded('subTasks')),
            // The derived completion ratio (an Eloquent accessor on the model).
            'progress' => $this->progress,
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }
}
