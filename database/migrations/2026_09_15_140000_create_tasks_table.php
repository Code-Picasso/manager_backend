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
            // A UUID primary key (matching the Flutter app's string ids) rather
            // than the usual auto-incrementing integer.
            $table->uuid('id')->primary();

            // foreignId() creates an unsigned big-int column matching users.id,
            // then constrained() turns it into a foreign key. cascadeOnDelete()
            // means deleting a user deletes their tasks automatically.
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();

            $table->string('title');
            $table->text('description')->default('');

            // Stored as strings; the Task model casts these to backed enums.
            $table->string('category')->default('work');
            $table->string('status')->default('today');

            // The due date (date only). Start/end times are "HH:MM" strings,
            // exactly as the Flutter app stores them.
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
