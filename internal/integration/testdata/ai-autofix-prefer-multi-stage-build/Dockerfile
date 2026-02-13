FROM golang:1.22-alpine
WORKDIR /src
COPY . .
RUN go build -o /out/app ./cmd/app
CMD ["app"]
