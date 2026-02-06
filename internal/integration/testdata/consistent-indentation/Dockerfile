ARG GO_VERSION=1.23
FROM golang:${GO_VERSION} AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app ./cmd/server

FROM alpine:3.20 AS runtime
RUN apk --no-cache add ca-certificates tzdata
USER nobody:nobody

FROM runtime
COPY --from=builder /app /usr/local/bin/app
EXPOSE 8080
ENTRYPOINT ["app"]
