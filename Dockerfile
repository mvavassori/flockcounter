# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Final stage
FROM debian:bookworm-slim AS runner

WORKDIR /app

# Install CA certificates for TLS verification
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/main .

# Create directory and copy MMDB file
RUN mkdir -p /app/data/geoip
COPY GeoLite2-City.mmdb /app/data/geoip/

# Expose the port your app runs on
EXPOSE 8080

# set the MMDB path env variable
ENV GEOIP_DB_PATH=/app/data/geoip/GeoLite2-City.mmdb

CMD ["./main"]
