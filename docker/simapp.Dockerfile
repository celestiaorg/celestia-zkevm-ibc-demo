# Stage 1: Generate the simapp binary (simd)
FROM --platform=$BUILDPLATFORM docker.io/golang:1.23.1-alpine3.20 AS builder

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GO111MODULE=on
# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
    gcc \
    git \
    # linux-headers are needed for Ledger support
    linux-headers \
    make \
    musl-dev
COPY . /celestia_zkevm_ibc_demo
WORKDIR /celestia_zkevm_ibc_demo

RUN uname -a &&\
    CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    make build-simapp

# Stage 2: Create a minimal image with just the binary
FROM docker.io/alpine:3.20

# Use UID 10,001 because UIDs below 10,000 are a security risk.
# Ref: https://github.com/hexops/dockerfile/blob/main/README.md#do-not-use-a-uid-below-10000
ARG UID=10001
ARG USER_NAME=celestia

ENV CELESTIA_HOME=/home/${USER_NAME}

# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
    bash \
    curl \
    jq \
    make \
    # Creates a user with $UID and $GID=$UID
    && adduser ${USER_NAME} \
    -D \
    -g ${USER_NAME} \
    -h ${CELESTIA_HOME} \
    -s /sbin/nologin \
    -u ${UID}

# Copy in the simd binary
COPY --from=builder /celestia_zkevm_ibc_demo/build/simd /bin/simd

USER ${USER_NAME}

# Set the working directory to the home directory.
WORKDIR ${CELESTIA_HOME}

CMD [ "/bin/simd" ]
