#!/bin/sh
# App container entrypoint — runs on every `docker compose up`.

set -e

# Poll until MySQL accepts connections (it may still be initialising).
until php -r "new PDO('mysql:host=' . getenv('DB_HOST') . ';dbname=' . getenv('DB_DATABASE'), getenv('DB_USERNAME'), getenv('DB_PASSWORD'));" 2>/dev/null; do
    echo "Waiting for MySQL at ${DB_HOST}..."
    sleep 2
done

echo "MySQL is up. Running migrations..."
php artisan migrate --force --no-interaction

echo "Starting the API server on http://localhost:8000 ..."
exec php artisan serve --host=0.0.0.0 --port=8000
