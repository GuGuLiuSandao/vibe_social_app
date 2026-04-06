FROM docker.m.daocloud.io/library/golang:1.25.3 AS builder
WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/api ./cmd/api
RUN mkdir -p /app/uploads

FROM docker.m.daocloud.io/library/alpine:3.20
WORKDIR /app
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/uploads /app/uploads
EXPOSE 8080
ENTRYPOINT ["/app/api"]
