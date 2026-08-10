# syntax=docker/dockerfile:1@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89
FROM ruby:3.3-slim@sha256:65bce1aa8dfc8758b8fadc4629a91d4cf621784b8dfb5fed265d6ed52b0a0fb3 AS build
WORKDIR /rails
COPY Gemfile Gemfile.lock ./
RUN bundle install
COPY . .
RUN bin/rails assets:precompile

FROM ruby:3.3-slim@sha256:65bce1aa8dfc8758b8fadc4629a91d4cf621784b8dfb5fed265d6ed52b0a0fb3 AS production
WORKDIR /rails
COPY --from=build /rails /rails
CMD ["bin/rails", "server"]
