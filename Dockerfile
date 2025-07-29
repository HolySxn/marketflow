FROM golang:1.24.5-bookworm AS builder
ARG CMD=marketflow

WORKDIR /build 
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /marketflow ./cmd/marketflow

# -- Final Stage ---
# Use a minimal base image for the final build
FROM gcr.io/distroless/static-debian12

COPY --from=builder /marketflow /marketflow
COPY configs/ /configs/

EXPOSE 8000

ENTRYPOINT ["/marketflow", "--config-path=/configs/config.yaml"]