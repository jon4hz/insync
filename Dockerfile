
FROM golang:1.17-alpine as builder

WORKDIR /app

RUN apk update --no-cache && \
    apk add gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o insync .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/insync .
CMD [ "./insync" ]