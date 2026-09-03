# syntax=docker/dockerfile:1@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32
FROM composer:2@sha256:d8f6343d3fae98107426bc49163ccad46ef85aabd4a27d80a74401fab4aba332 AS dependencies
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-interaction

FROM php:8.4-fpm@sha256:59fa733c9af643a122f8a9976119460e35ce76dd0a3f2b9c8f75af8e361a54e2 AS production
WORKDIR /app
COPY --from=dependencies /app/vendor ./vendor
RUN pecl install xdebug
COPY . .
CMD ["php-fpm"]
