# syntax=docker/dockerfile:1.6

FROM ubuntu:22.04

ARG ARDUPILOT_REF=master
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Etc/UTC
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
    tzdata \
    wget \
    && ln -snf "/usr/share/zoneinfo/${TZ}" /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt
RUN git clone --depth 1 --branch "${ARDUPILOT_REF}" https://github.com/ArduPilot/ardupilot.git /opt/ardupilot

WORKDIR /opt/ardupilot
RUN git submodule update --init --recursive --depth 1

RUN useradd -m -s /bin/bash ardupilot \
    && echo "ardupilot ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/ardupilot \
    && echo 'Defaults env_keep += "DEBIAN_FRONTEND TZ"' > /etc/sudoers.d/ardupilot-env \
    && chmod 0440 /etc/sudoers.d/ardupilot \
    && chmod 0440 /etc/sudoers.d/ardupilot-env \
    && chown -R ardupilot:ardupilot /opt/ardupilot
ENV USER=ardupilot
ENV LOGNAME=ardupilot
USER ardupilot
RUN Tools/environment_install/install-prereqs-ubuntu.sh -y
ENV PATH="/home/ardupilot/.local/bin:/usr/lib/ccache:/opt/gcc-arm-none-eabi-10-2020-q4-major/bin:/opt/ardupilot/Tools/autotest:${PATH}"
RUN ./waf configure --board sitl \
    && ./waf build --target bin/ardurover

ENTRYPOINT ["sim_vehicle.py"]
CMD ["-v", "Rover", "-f", "rover", "--no-mavproxy", "--speedup=1", "--aircraft", "semlink-ardurover", "-A", "--serial0=udpclient:semlink:14550", "-w"]
