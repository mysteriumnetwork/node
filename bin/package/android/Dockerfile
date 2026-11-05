FROM golang:1.26.10-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends openjdk-17-jdk-headless unzip \
    && rm -rf /var/lib/apt/lists/*

ENV ANDROID_HOME=/opt/android-sdk
ARG CMDLINE_TOOLS=commandlinetools-linux-15859902_latest.zip
ARG CMDLINE_TOOLS_SHA256=4e4c464f145a7512b57d088ac6c278c03c9eea610886b35a5e0804e74eedf583
ARG NDK_VERSION=27.1.12297006

RUN mkdir -p $ANDROID_HOME/cmdline-tools \
    && curl -fsSL -o /tmp/clt.zip https://dl.google.com/android/repository/$CMDLINE_TOOLS \
    && echo "$CMDLINE_TOOLS_SHA256  /tmp/clt.zip" | sha256sum -c - \
    && unzip -q /tmp/clt.zip -d $ANDROID_HOME/cmdline-tools \
    && mv $ANDROID_HOME/cmdline-tools/cmdline-tools $ANDROID_HOME/cmdline-tools/latest \
    && rm /tmp/clt.zip \
    && yes | $ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager --licenses >/dev/null \
    && $ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager "ndk;$NDK_VERSION" "platforms;android-35" >/dev/null

RUN git config --global --add safe.directory /src
WORKDIR /src
