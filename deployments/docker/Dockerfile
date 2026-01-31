# Stage 1: Build ncnn
FROM ubuntu:22.04 AS ncnn-builder

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    libvulkan-dev \
    libwebp-dev \
    glslang-tools \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Force git to use HTTPS instead of SSH for submodules
RUN git config --global url."https://github.com/".insteadOf git@github.com:

# Clone Real-ESRGAN-ncnn-vulkan
# Using specific tag for stability or latest
RUN git clone --recursive https://github.com/xinntao/Real-ESRGAN-ncnn-vulkan.git

WORKDIR /build/Real-ESRGAN-ncnn-vulkan/src

# Build with optimizations
RUN cmake -B build \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS="-O3" \
    -DCMAKE_C_FLAGS="-O3" && \
    cmake --build build -j$(nproc)

# Stage 2: Build Go application
FROM golang:1.24-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o upscale-server \
    ./cmd/server

# Stage 3: Runtime
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libvulkan1 \
    mesa-vulkan-drivers \
    libwebp7 \
    imagemagick \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Create app user
RUN useradd -m -u 1000 appuser

WORKDIR /app

# Copy binaries
COPY --from=ncnn-builder /build/Real-ESRGAN-ncnn-vulkan/src/build/realesrgan-ncnn-vulkan ./bin/
COPY --from=go-builder /app/upscale-server ./

# Copy models (we assume they are in the build context or we download them here? 
# Better to copy from context if "download-models.sh" was run before build, OR download in a stage)
# The plan says "COPY models ./models/" implying they are local.
COPY models ./models/
COPY config ./config/
COPY docs ./docs/

# Create data directories
RUN mkdir -p data/uploads data/outputs && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8089

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8089/api/v1/health || exit 1

CMD ["./upscale-server", "-config", "config/config.docker.yaml"]
