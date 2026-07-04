FROM golang:1.26-alpine AS builder

WORKDIR /app

# Load dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy files
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o main .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .
EXPOSE 8080

CMD ["./main"]
