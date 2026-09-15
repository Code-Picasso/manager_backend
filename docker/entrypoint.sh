#!/bin/sh
# Entrypoint for the app container. Runs on every `docker compose up`.

set -e

# Wait until MySQL actually accepts connections. The app container may start
# before MySQL has finished initialising, so we poll with a tiny PDO probe.
until php -r "new PDO('mysql:host=' . getenv('DB_HOST') . ';dbname=' . getenv('DB_DATABASE'), getenv('DB_USERNAME'), getenv('DB_PASSWORD'));" 2>/dev/null; do
    echo "Waiting for MySQL at ${DB_HOST}..."
    sleep 2
done

echo "MySQL is up. Running migrations..."
php artisan migrate --force --no-interaction

echo "Starting the API server on http://localhost:8000 ..."
exec php artisan serve --host=0.0.0.0 --port=8000
