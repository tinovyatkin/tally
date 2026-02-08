FROM ubuntu:22.04
COPY --chmod=0755 <<EOF /entrypoint.sh
#!/bin/sh
EOF
