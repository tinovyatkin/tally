# syntax=docker/dockerfile:1
FROM ruby:3.3-slim AS build
WORKDIR /rails
COPY Gemfile Gemfile.lock ./
RUN bundle install
COPY . .
RUN bin/rails assets:precompile

FROM ruby:3.3-slim AS production
WORKDIR /rails
COPY --from=build /rails /rails
CMD ["bin/rails", "server"]
