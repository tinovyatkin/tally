FROM ubuntu:22.04
RUN apt-get update
SHELL ["/bin/bash", "-c"]
RUN <<EOF
set -e
echo doneRUN apt-get update
ln -sf /bin/bash /bin/sh
echo done
EOF
