# Build stage
FROM golang:1.21 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY ./resourses/ ./resourses/
COPY ./util/ ./util/
COPY ./main.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o aws-mfa-go main.go

# Final stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/aws-mfa-go .

# Command to run the executable
ENTRYPOINT ["./aws-mfa-go"]