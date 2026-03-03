FROM golang:1.21-alpine

WORKDIR /app

# Pehle modules setup karenge
COPY go.mod ./
# Forcefully dependencies install karne ke liye
RUN go mod tidy

# Sari files copy karo
COPY . .

# Environment setup for build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

EXPOSE 8080

CMD ["./main"]
