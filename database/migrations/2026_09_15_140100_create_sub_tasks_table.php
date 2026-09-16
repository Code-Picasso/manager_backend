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

            // Owning task; deleting the task deletes its sub-tasks.
            $table->foreignUuid('task_id')->constrained()->cascadeOnDelete();

            $table->string('title');
            $table->boolean('is_done')->default(false);

            // Ordering position within the checklist.
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
