FROM golang:1.23.8 AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./
# COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o firestore-admin-client .

# Create a minimal production image
FROM debian:bookworm-slim


# Set working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/firestore-admin-client .

# Copy Firebase credentials
# COPY firebase-credential.json .

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./firestore-admin-client"]
