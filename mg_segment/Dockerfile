# Build stage: Go + Tesseract dev libraries for CGO compilation
FROM golang:1.21-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
        tesseract-ocr \
        tesseract-ocr-eng \
        libtesseract-dev \
        libleptonica-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=1 go build -o mg_segment .

# Runtime stage: smaller image with only the Tesseract runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        tesseract-ocr \
        tesseract-ocr-eng \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/mg_segment /usr/local/bin/mg_segment

# /input  → mount your WhatsApp JPEG source directory here
# /output → mount your output directory here
ENTRYPOINT ["mg_segment"]
CMD ["/input", "/output"]
