# Stage 1: Build the Go application
FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Stage 2: Create a minimal image
FROM alpine:latest

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy .env file
COPY .env .

# Expose port
EXPOSE 50051

# Command to run the executable
CMD ["./main"]
