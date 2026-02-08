FROM ubuntu:22.04
WORKDIR /app
RUN make build
RUN apt-get install curl
