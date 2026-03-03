FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

EXPOSE 8080

CMD ["./main"]
