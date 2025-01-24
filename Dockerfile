# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23.4-bookworm AS builder
ARG TARGETARCH
ARG TARGETOS

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod tidy

# Copy the source from the current directory to the Working Directory inside the container
COPY ./resourses/ ./resourses/
COPY ./util/ ./util/
COPY ./main.go .

# Build the Go app with detailed error output
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH GOOS=linux go build -o aws-mfa-go -a -ldflags="-s -w" -installsuffix cgo || { echo 'Go build failed'; exit 1; }

# Final stage
FROM scratch AS final

WORKDIR /app

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
LABEL org.opencontainers.image.source="https://github.com/bariiss/aws-mfa-go"

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/aws-mfa-go .

# Command to run the executable
ENTRYPOINT ["/app/aws-mfa-go"]