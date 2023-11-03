FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go mod verify && GOOS=linux GOARCH=386 go build -o /app/spirit

FROM scratch

WORKDIR /app

COPY --from=builder /app/spirit /app/spirit
COPY --from=builder /app/db /app/
COPY --from=builder /app/db/ /app/

EXPOSE 3000

ENTRYPOINT ["/app/spirit"]