<?php

namespace App\Http\Requests\Concerns;

/** Keeps empty text fields as empty strings rather than null. */
trait NormalizesEmptyText
{
    /** Rewrite null fields in $fields to empty strings. @param  list<string>  $fields */
    protected function normalizeEmptyText(array $fields): void
    {
        $merge = [];

        foreach ($fields as $field) {
            if ($this->has($field) && $this->input($field) === null) {
                $merge[$field] = '';
            }
        }

        if ($merge !== []) {
            $this->merge($merge);
        }
    }
}
