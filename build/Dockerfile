FROM sc4h/alpine-s6overlay:3.13

ENV \
  CONFIG_DIR=/config

# Binary
COPY ["dist/tqm_linux_amd64/tqm", "/app/tqm/tqm"]

# Add root files
COPY ["build/root/", "/"]

EXPOSE 7337

VOLUME ["/config"]
