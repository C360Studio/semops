# syntax=docker/dockerfile:1.6
#
# Build:
#
#   docker build \
#     --platform linux/amd64 \
#     -f docker/ardupilot-sitl/Dockerfile \
#     -t c360studio/semops-ardupilot-sitl:local \
#     docker/ardupilot-sitl
#
# This image is the preferred first ArduPilot source for the SemOps
# ardupilot-stack gate. It packages the official ArduCopter Linux SITL binary
# without Gazebo so the first parity proof stays focused on ArduPilot MAVLink
# telemetry through SemOps.

ARG UBUNTU_BASE=ubuntu:22.04
ARG UBUNTU_BASE_PLATFORM=linux/amd64

FROM --platform=${UBUNTU_BASE_PLATFORM} ${UBUNTU_BASE} AS runtime

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG ARDUPILOT_REF=918718f6b063cca9a60de3921c3dcee2e8ca3524
ARG ARDUPILOT_VERSION=4.8.0-dev
ARG ARDUPILOT_BINARY_URL=https://firmware.ardupilot.org/Copter/latest/SITL_x86_64_linux_gnu/arducopter
ARG ARDUPILOT_BINARY_SHA256=1646845efcf96b196ad2f39f186a5b9dd325877b938245e7374d2dbc8a29c9bd

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      python3-pip \
      tini \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "${ARDUPILOT_BINARY_URL}" -o /usr/local/bin/arducopter \
    && echo "${ARDUPILOT_BINARY_SHA256}  /usr/local/bin/arducopter" | sha256sum -c - \
    && chmod +x /usr/local/bin/arducopter \
    && python3 -m pip install --no-cache-dir MAVProxy==1.8.74 future==1.0.0 \
    && mkdir -p /var/lib/semops-ardupilot

COPY entrypoint.sh /usr/local/bin/semops-ardupilot-sitl
RUN chmod +x /usr/local/bin/semops-ardupilot-sitl

LABEL org.opencontainers.image.title="SemOps ArduPilot SITL" \
      org.opencontainers.image.description="Preferred ArduPilot SITL source for SemOps ardupilot-stack evidence" \
      org.opencontainers.image.source="https://github.com/C360Studio/semops" \
      org.opencontainers.image.vendor="C360Studio" \
      org.c360.semops.ardupilot.ref="${ARDUPILOT_REF}" \
      org.c360.semops.ardupilot.version="${ARDUPILOT_VERSION}" \
      org.c360.semops.ardupilot.binary.url="${ARDUPILOT_BINARY_URL}" \
      org.c360.semops.ardupilot.binary.sha256="${ARDUPILOT_BINARY_SHA256}"

WORKDIR /var/lib/semops-ardupilot
ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/semops-ardupilot-sitl"]
