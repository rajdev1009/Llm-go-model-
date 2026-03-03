FROM golang:1.21-alpine

WORKDIR /app

# Pehle dependencies setup karenge
COPY go.mod ./
# go.sum agar nahi bhi hai toh ye command use generate kar degi
RUN go mod tidy

# Ab baki sari files copy karenge
COPY . .

# Build process
RUN go build -o main .

EXPOSE 8080

# Application start
CMD ["./main"]
