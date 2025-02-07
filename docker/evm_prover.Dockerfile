# Use official Rust image as base
FROM rust:1.83-slim-bookworm AS builder

# Install dependencies
RUN apt-get update && apt-get install -y \
    git \
    pkg-config \
    libssl-dev \
    build-essential \
    curl \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Clone SP1 repo and install succinct toolchain
RUN mkdir -p /sp1
ENV SP1_DIR=/sp1
WORKDIR $SP1_DIR

RUN curl -Lv https://sp1.succinct.xyz | bash -x
RUN $SP1_DIR/bin/sp1up

# Clone the repo and build
WORKDIR /celestia_zkevm_ibc_demo/
COPY . .

# Build release binary
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libssl3 \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /celestia_zkevm_ibc_demo/target/release/evm-prover /usr/local/bin/

# Create non-root user
RUN useradd -m -u 10001 -s /bin/bash prover

USER prover
WORKDIR /home/prover

COPY --from=builder /celestia_zkevm_ibc_demo/provers/evm-prover/proto_descriptor.bin .

# Default environment variables that can be overridden
ENV TENDERMINT_RPC_URL=http://localhost:5123
ENV RPC_URL=http://localhost:8545
ENV CONTRACT_ADDRESS=0x2854CFaC53FCaB6C95E28de8C91B96a31f0af8DD
ENV PROVER_PORT=50052

# Expose port
EXPOSE ${PROVER_PORT}

# Run the prover
CMD ["evm-prover"]
