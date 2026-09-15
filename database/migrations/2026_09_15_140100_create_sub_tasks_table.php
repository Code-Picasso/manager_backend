<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('sub_tasks', function (Blueprint $table) {
            $table->uuid('id')->primary();

            // foreignUuid() creates a uuid column constrained to tasks.id, and
            // cascadeOnDelete() removes sub-tasks when their task is deleted.
            $table->foreignUuid('task_id')->constrained()->cascadeOnDelete();

            $table->string('title');
            $table->boolean('is_done')->default(false);

            // Explicit ordering so the checklist preserves the client's order
            // even when several sub-tasks are inserted in the same second.
            $table->unsignedInteger('position')->default(0);

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('sub_tasks');
    }
};
