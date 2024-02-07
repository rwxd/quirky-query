FROM docker.io/golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o quirky-query

FROM docker.io/alpine:latest

WORKDIR /app

COPY --from=builder /app/templates ./templates
COPY --from=builder /app/quirky-query .


ENTRYPOINT ["./quirky-query"]
CMD []
