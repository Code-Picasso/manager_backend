<?php

namespace App\Http\Controllers;

use App\Http\Requests\StoreNoteRequest;
use App\Http\Requests\UpdateNoteRequest;
use App\Http\Resources\NoteResource;
use App\Models\Note;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;
use Illuminate\Http\Response;

/** CRUD for notes. */
class NoteController extends Controller
{
    /** List the user's notes, most recently updated first. */
    public function index(Request $request): AnonymousResourceCollection
    {
        $notes = $request->user()
            ->notes()
            ->latest('updated_at')
            ->get();

        return NoteResource::collection($notes);
    }

    /** Create a note. */
    public function store(StoreNoteRequest $request): JsonResponse
    {
        $note = $request->user()->notes()->create($request->validated());

        return (new NoteResource($note))->response()->setStatusCode(201);
    }

    /** Show a single note. */
    public function show(Request $request, Note $note): NoteResource
    {
        abort_unless($note->user_id === $request->user()->id, 404);

        return new NoteResource($note);
    }

    /** Update a note (title and/or content). */
    public function update(UpdateNoteRequest $request, Note $note): NoteResource
    {
        abort_unless($note->user_id === $request->user()->id, 404);

        $note->update($request->validated());

        return new NoteResource($note->fresh());
    }

    /** Delete a note. */
    public function destroy(Request $request, Note $note): Response
    {
        abort_unless($note->user_id === $request->user()->id, 404);

        $note->delete();

        return response()->noContent();
    }
}
