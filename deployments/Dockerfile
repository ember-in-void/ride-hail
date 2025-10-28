FROM golang:1.24-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/ride-hail-system .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/ride-hail-system ./ride-hail-system
ENV CONFIG_DIR=/app/config
CMD ["./ride-hail-system", "-service=all"]