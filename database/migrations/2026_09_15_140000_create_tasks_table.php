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
        Schema::create('tasks', function (Blueprint $table) {
            // UUID primary key.
            $table->uuid('id')->primary();

            // Owning user; deleting the user deletes their tasks.
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();

            $table->string('title');
            // MySQL cannot default a TEXT column; Task::$attributes supplies the ''.
            $table->text('description');

            // Stored as strings; the Task model casts these to backed enums.
            $table->string('category')->default('work');
            $table->string('status')->default('today');

            // Due date, with "HH:MM" start and end times.
            $table->date('date');
            $table->string('start_time')->default('09:00');
            $table->string('end_time')->default('10:00');

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('tasks');
    }
};
