# ---------------------------------------------------------------------------
# Build stage for the Laravel API application.
#
# This is a single-stage image that runs the app with PHP's built-in
# development server (`php artisan serve`). It is deliberately simple so it is
# easy to read and learn from; the README explains the nginx + PHP-FPM layout
# you would use in production.
# ---------------------------------------------------------------------------

# Official PHP CLI image. Alpine keeps the image small; the version matches
# Laravel 12's PHP 8.2+ requirement.
FROM php:8.4-cli-alpine

# Install the build toolchain, compile the `pdo_mysql` extension (Laravel needs
# it to talk to MySQL), then remove the build tools to keep the image lean.
# `$PHPIZE_DEPS` is a helper variable provided by the official PHP images.
RUN apk add --no-cache --virtual .build-deps $PHPIZE_DEPS \
    && docker-php-ext-install pdo_mysql \
    && apk del .build-deps \
    && apk add --no-cache git unzip

# Copy the Composer binary out of its own official image (a "multi-stage" COPY).
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /var/www/html

# Copy the application source. `vendor/` and `.env` are excluded via
# .dockerignore — dependencies are installed fresh, and configuration comes
# from environment variables supplied by docker-compose.
COPY . .

# Install production dependencies (no dev tooling, and an optimised autoloader).
RUN composer install --no-dev --optimize-autoloader --no-interaction

# The server runs as www-data; give it write access to Laravel's cache/log dirs.
RUN chown -R www-data:www-data storage bootstrap/cache

# The API dev server listens on this port inside the container.
EXPOSE 8000

# Our entrypoint waits for MySQL, runs migrations, then starts the server.
COPY docker/entrypoint.sh /usr/local/bin/entrypoint
RUN chmod +x /usr/local/bin/entrypoint

ENTRYPOINT ["entrypoint"]
