FROM golang:1.21-alpine

WORKDIR /app

# Copy source
COPY go.mod main.go ./

# Build
RUN go build -o mg_segment .

# Create output directory
RUN mkdir -p /output

ENTRYPOINT ["/app/mg_segment"]
CMD ["/input", "/output"]
