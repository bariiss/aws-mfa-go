FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /app

# Copy go.mod/go.sum first to cache modules more effectively
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your application code
COPY ./resourses/ ./resourses/
COPY ./util/ ./util/
COPY ./main.go .

# Build the Go app with detailed output
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -o aws-mfa-go -a -ldflags="-s -w" -installsuffix cgo

# Final Stage
FROM scratch AS final
WORKDIR /app

# Copy CA certs (make sure they’re installed in the builder!)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
LABEL org.opencontainers.image.source="https://github.com/bariiss/aws-mfa-go"

# Copy built binary from the builder stage
COPY --from=builder /app/aws-mfa-go .

# Run
ENTRYPOINT ["/app/aws-mfa-go"]