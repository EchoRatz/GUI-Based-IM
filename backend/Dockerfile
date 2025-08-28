# backend/Dockerfile
FROM golang:1.25
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build whole package (includes db.go)
RUN go build -o server .

EXPOSE 8080
CMD ["./server"]
