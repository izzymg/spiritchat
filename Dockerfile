FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go mod verify && GOOS=linux GOARCH=386 go build -o /app/spirit

FROM debian

WORKDIR /app

COPY --from=builder app/spirit /app/spirit
COPY --from=builder app/db/ /app/db/
RUN ls /app/db

EXPOSE 3000

ENTRYPOINT ["/app/spirit"]