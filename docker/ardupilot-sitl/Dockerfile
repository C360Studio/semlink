# syntax=docker/dockerfile:1.6

FROM ubuntu:22.04

ARG ARDUPILOT_REF=master
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/opt/ardupilot/Tools/autotest:${PATH}"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    python3 \
    python3-dev \
    python3-pip \
    python3-setuptools \
    python3-wheel \
    sudo \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt
RUN git clone --depth 1 --branch "${ARDUPILOT_REF}" https://github.com/ArduPilot/ardupilot.git /opt/ardupilot

WORKDIR /opt/ardupilot
RUN git submodule update --init --recursive --depth 1
RUN Tools/environment_install/install-prereqs-ubuntu.sh -y

ENTRYPOINT ["sim_vehicle.py"]
CMD ["-v", "Rover", "-f", "rover", "--no-mavproxy", "--speedup=1", "--aircraft", "semlink-ardurover", "-A", "--serial0=udpclient:semlink:14550", "-w"]
