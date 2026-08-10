# syntax=docker/dockerfile:1@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89
FROM composer:2@sha256:4d71c3c2109c61d5415544264b59ad4087e4c5b7244481723664138fd36d5040 AS dependencies
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-interaction

FROM php:8.4-fpm@sha256:9467f10bf42897dec0abb73ee20c747ebd45463ec9d6fcc4044cb83eba6dade7 AS production
WORKDIR /app
COPY --from=dependencies /app/vendor ./vendor
RUN pecl install xdebug
COPY . .
CMD ["php-fpm"]
