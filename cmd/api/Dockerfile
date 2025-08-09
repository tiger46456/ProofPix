# Build stage
FROM golang:1.23-alpine AS build

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the api binary with CGO disabled
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Final stage
FROM alpine:latest

# Copy the compiled binary from build stage
COPY --from=build /app/api /usr/local/bin/api

# Expose port 8080
EXPOSE 8080

# Set the command to run the binary
CMD ["api"]