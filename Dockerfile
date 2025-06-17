# Start from official Go base image
FROM golang:1.24.3-bullseye

# Set working directory
WORKDIR /app

# Copy go.mod and source code
COPY go.mod ./
COPY . .

# Build the Go app
RUN go build -o otp-service

# Run the service
CMD ["./otp-service"]