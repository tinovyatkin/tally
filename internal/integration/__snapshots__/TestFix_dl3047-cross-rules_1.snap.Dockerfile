FROM ubuntu:22.04
ADD --unpack http://example.com/archive.tar.gz /opt
RUN wget --progress=dot:giga http://example.com/archive.tar.gz | tar -xz -C /opt
RUN wget http://example.com/config.json -O /etc/app/config.json
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN curl -fsSL http://example.com/script.sh | sh
