FROM golang:1.21-alpine

WORKDIR /app

# 1. Pehle saari files copy karo (taaki main.go mil jaye)
COPY . .

# 2. Ab dependencies ko tidy karo (ye go.sum khud bana dega)
RUN go mod tidy

# 3. Ab build karo
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

EXPOSE 8080

CMD ["./main"]
