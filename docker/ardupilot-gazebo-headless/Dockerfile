# syntax=docker/dockerfile:1.6
#
# Build:
#
#   docker build \
#     -f docker/ardupilot-gazebo-headless/Dockerfile \
#     -t c360studio/semops-ardupilot-gazebo-headless:local \
#     docker/ardupilot-gazebo-headless
#
# This image is an opt-in ArduPilot/Gazebo source for the SemOps MAVLink
# ardupilot-stack gate. It is not a default pull path.

ARG ARDUPILOT_DEV_BASE=ardupilot/ardupilot-dev-base:v0.2.0

FROM ${ARDUPILOT_DEV_BASE} AS runtime

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG ARDUPILOT_REF=918718f6b063cca9a60de3921c3dcee2e8ca3524
ARG ARDUPILOT_GAZEBO_REF=082a0fe231f6e63bc8d1598f1cba461d9e2ea7f5
ARG GZ_VERSION=harmonic
ARG GZ_PACKAGE=gz-harmonic

ENV DEBIAN_FRONTEND=noninteractive
ENV GZ_VERSION=${GZ_VERSION}
ENV LIBGL_ALWAYS_SOFTWARE=1
ENV QT_QPA_PLATFORM=offscreen
ENV GZ_SIM_SYSTEM_PLUGIN_PATH=/opt/ardupilot_gazebo/build
ENV GZ_SIM_RESOURCE_PATH=/opt/ardupilot_gazebo/models:/opt/ardupilot_gazebo/worlds
ENV PATH=/opt/ardupilot/Tools/autotest:${PATH}

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      gnupg \
      lsb-release \
    && curl -fsSL https://packages.osrfoundation.org/gazebo.gpg \
      --output /usr/share/keyrings/pkgs-osrf-archive-keyring.gpg \
    && printf 'deb [arch=%s signed-by=%s] %s %s main\n' \
      "$(dpkg --print-architecture)" \
      "/usr/share/keyrings/pkgs-osrf-archive-keyring.gpg" \
      "https://packages.osrfoundation.org/gazebo/ubuntu-stable" \
      "$(lsb_release -cs)" \
      > /etc/apt/sources.list.d/gazebo-stable.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
      ${GZ_PACKAGE} \
      build-essential \
      cmake \
      gstreamer1.0-gl \
      gstreamer1.0-libav \
      gstreamer1.0-plugins-bad \
      libgstreamer-plugins-base1.0-dev \
      libgstreamer1.0-dev \
      libgz-sim8-dev \
      libopencv-dev \
      pkg-config \
      rapidjson-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt

RUN git clone https://github.com/ArduPilot/ardupilot.git ardupilot \
    && cd ardupilot \
    && git checkout "${ARDUPILOT_REF}" \
    && git submodule update --init --recursive \
    && ./waf configure --board sitl \
    && ./waf copter \
    && ln -sf /opt/ardupilot/Tools/autotest/sim_vehicle.py /usr/local/bin/sim_vehicle.py

RUN git clone https://github.com/ArduPilot/ardupilot_gazebo.git ardupilot_gazebo \
    && cd ardupilot_gazebo \
    && git checkout "${ARDUPILOT_GAZEBO_REF}" \
    && cmake -S . -B build -DCMAKE_BUILD_TYPE=RelWithDebInfo \
    && cmake --build build -j"$(nproc)"

COPY entrypoint.sh /usr/local/bin/semops-ardupilot-gazebo-headless
RUN chmod +x /usr/local/bin/semops-ardupilot-gazebo-headless

LABEL org.opencontainers.image.title="SemOps ArduPilot Gazebo headless SITL" \
      org.opencontainers.image.description="Opt-in ArduPilot/Gazebo source for SemOps ardupilot-stack evidence" \
      org.opencontainers.image.source="https://github.com/C360Studio/semops" \
      org.opencontainers.image.vendor="C360Studio" \
      org.c360.semops.ardupilot.ref="${ARDUPILOT_REF}" \
      org.c360.semops.ardupilot_gazebo.ref="${ARDUPILOT_GAZEBO_REF}" \
      org.c360.semops.gz.version="${GZ_VERSION}"

CMD ["/usr/local/bin/semops-ardupilot-gazebo-headless"]
