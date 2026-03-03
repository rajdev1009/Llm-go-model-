FROM golang:1.21-alpine
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY *.go ./
COPY bot_instructions.txt ./
RUN go build -o main .
EXPOSE 8080
CMD ["./main"]
