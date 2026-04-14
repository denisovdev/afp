FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/server /bin/server
COPY migrations /migrations

ENV MIGRATIONS_DIR=/migrations

EXPOSE 8080
ENTRYPOINT ["/bin/server"]
