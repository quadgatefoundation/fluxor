FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy root module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the load-balancing example
RUN go build -o load-balancing ./examples/load-balancing/main.go

FROM alpine:latest

WORKDIR /app

# Create examples/load-balancing directory structure
RUN mkdir -p examples/load-balancing

# Copy config file to maintain path structure
COPY --from=builder /app/examples/load-balancing/config.json ./examples/load-balancing/

# Copy binary
COPY --from=builder /app/load-balancing ./

EXPOSE 8080 9090

CMD ["./load-balancing"]

