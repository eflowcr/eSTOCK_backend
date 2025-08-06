FROM golang:1.23.0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY cmd/.env .env

RUN go build -o main ./cmd

CMD ["./main"]