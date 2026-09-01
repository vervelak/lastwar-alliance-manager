# Build stage
FROM golang:1.27-alpine AS builder

# Install build dependencies (gcc/g++/musl for CGO; tesseract-ocr-dev for gosseract)
RUN apk add --no-cache gcc g++ musl-dev tesseract-ocr-dev leptonica-dev

WORKDIR /app

# Copy go mod files and download dependencies first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for gosseract/Tesseract
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Runtime stage
FROM alpine:3.24

# Install runtime dependencies:
#   ca-certificates  - HTTPS support
#   tesseract-ocr    - Tesseract OCR engine (required by gosseract)
#   tesseract-ocr-data-eng - English language data for OCR
#   tesseract-ocr-data-deu - German language data for OCR
RUN apk add --no-cache ca-certificates tesseract-ocr tesseract-ocr-data-eng tesseract-ocr-data-deu

WORKDIR /app

# Copy binary and static assets from builder
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static

# Create data directory for the SQLite database (mounted as a volume)
RUN mkdir -p /data

# Expose the application port
EXPOSE 8080

# Database stored on the mounted volume; SESSION_KEY should be set at runtime
ENV DATABASE_PATH=/data/alliance.db

# Run the application
CMD ["./main"]
