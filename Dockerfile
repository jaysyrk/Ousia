FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download || true

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/gateway ./cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/sidecar ./cmd/sidecar
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ousiactl ./cmd/ousiactl

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl tzdata
WORKDIR /app

COPY --from=builder /bin/gateway /usr/local/bin/gateway
COPY --from=builder /bin/sidecar /usr/local/bin/sidecar
COPY --from=builder /bin/ousiactl /usr/local/bin/ousiactl

COPY certs /app/certs
COPY ousia.yaml /app/ousia.yaml

EXPOSE 8080 8443 9000

CMD ["gateway", "-config", "/app/ousia.yaml"]
