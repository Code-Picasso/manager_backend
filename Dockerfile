# Single-stage image running the API with PHP's built-in dev server.

# PHP 8.4 CLI on Alpine — satisfies Laravel 12's PHP 8.2+ requirement.
FROM php:8.4-cli-alpine

# Compile the pdo_mysql extension MySQL needs, then drop the build toolchain.
RUN apk add --no-cache --virtual .build-deps $PHPIZE_DEPS \
    && docker-php-ext-install pdo_mysql \
    && apk del .build-deps \
    && apk add --no-cache git unzip

# Copy the Composer binary from its official image.
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /var/www/html

# Copy the app source — vendor/ and .env are excluded via .dockerignore.
COPY . .

# Install production dependencies with an optimised autoloader.
RUN composer install --no-dev --optimize-autoloader --no-interaction

# Give www-data write access to Laravel's cache and log directories.
RUN chown -R www-data:www-data storage bootstrap/cache

# The port the dev server listens on.
EXPOSE 8000

# Wait for MySQL, run migrations, then start the server.
COPY docker/entrypoint.sh /usr/local/bin/entrypoint
RUN chmod +x /usr/local/bin/entrypoint

ENTRYPOINT ["entrypoint"]
