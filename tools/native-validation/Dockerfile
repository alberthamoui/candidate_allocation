FROM golang:1.25.9-bookworm
RUN sed -i s,http://,https://,g /etc/apt/sources.list.d/debian.sources && apt-get update && apt-get install -y --no-install-recommends \
    pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev xvfb xauth dbus-x11 \
    xdotool scrot fonts-dejavu-core && rm -rf /var/lib/apt/lists/*
ARG VALIDATION_UID=1000
ARG VALIDATION_GID=1000
RUN groupadd --gid ${VALIDATION_GID} candidate && useradd --uid ${VALIDATION_UID} --gid candidate --create-home candidate
RUN apt-get update && apt-get install -y --no-install-recommends xclip && rm -rf /var/lib/apt/lists/*
WORKDIR /src
