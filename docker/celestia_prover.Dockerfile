# Use official Rust image as base
FROM rust:1.85-slim-bookworm AS builder

# Install dependencies
RUN apt-get update && apt-get install -y \
    git \
    pkg-config \
    libssl-dev \
    build-essential \
    curl \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Install SP1 toolchain
RUN curl -Lv https://sp1.succinct.xyz | bash -x
RUN bash -c 'source /root/.bashrc && sp1up'

RUN rustup toolchain list
RUN rustup default stable

# Install just
RUN cargo install just

# Copy the repo
WORKDIR /celestia_zkevm_ibc_demo/
COPY . .

# Build SP1 programs
WORKDIR /celestia_zkevm_ibc_demo/solidity-ibc-eureka
ENV PATH="/root/.sp1/bin:$PATH"
RUN bash -c 'source /root/.bashrc && just build-sp1-programs'

# Build celestia-prover binary
WORKDIR /celestia_zkevm_ibc_demo
RUN cargo build --bin celestia-prover --release --locked

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libssl3 \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /celestia_zkevm_ibc_demo/target/release/celestia-prover /usr/local/bin/

# Create non-root user
RUN useradd -m -u 10001 -s /bin/bash prover

USER prover
WORKDIR /home/prover

COPY --from=builder /celestia_zkevm_ibc_demo/provers/celestia-prover/proto_descriptor.bin .

# Default environment variables that can be overridden
ENV TENDERMINT_RPC_URL=http://localhost:5123
ENV RPC_URL=http://localhost:8545
ENV CONTRACT_ADDRESS=0x2854CFaC53FCaB6C95E28de8C91B96a31f0af8DD
ENV CELESTIA_PROTO_DESCRIPTOR_PATH=proto_descriptor.bin
ENV SP1_PROVER=mock
ENV PROVER_PORT=50051

# Expose port
EXPOSE ${PROVER_PORT}

# Run the prover
CMD ["celestia-prover"]
