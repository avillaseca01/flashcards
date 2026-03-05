### Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /build
RUN apk add --no-cache git

COPY go.mod main.go ./
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o flashcards .

### Stage 2: Final (~15MB)
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/flashcards .
COPY static/ ./static/

RUN mkdir -p /data

EXPOSE 8080

ENV PORT=8080
ENV DATA_DIR=/data

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/api/cards/stats || exit 1

CMD ["/app/flashcards"]