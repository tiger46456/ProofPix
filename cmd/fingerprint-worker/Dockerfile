# Build stage
FROM golang:1.23-alpine AS build

# Set working directory
WORKDIR /app

# Install C dependencies needed for FAISS compilation
RUN apk --no-cache add gcc g++ musl-dev pkgconfig openblas-dev

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
RUN apk --no-cache add build-base openblas-dev
COPY . .

# Build the fingerprint-worker binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o fingerprint-worker .

# Final stage
FROM alpine:latest

# Install C-based libraries needed for FAISS
RUN apk --no-cache add openblas libgomp

# Copy the compiled binary from build stage
COPY --from=build /app/fingerprint-worker /usr/local/bin/fingerprint-worker

# Set the command to run the binary
CMD ["fingerprint-worker"]