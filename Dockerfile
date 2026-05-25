FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/expense-manager ./cmd/server

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app && mkdir -p /app/data /app/public && chown -R app:app /app

COPY --from=builder /out/expense-manager /app/expense-manager

ENV PORT=3000 \
    STORE_DRIVER=mysql \
    MYSQL_DSN=expense:expense@tcp(mysql:3306)/expense_manager?charset=utf8mb4&parseTime=false&loc=Local \
    PUBLIC_DIR=/app/public

EXPOSE 3000

USER app

CMD ["/app/expense-manager"]
