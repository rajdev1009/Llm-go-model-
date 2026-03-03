FROM golang:1.21-alpine

WORKDIR /app

# Dependencies handle karna
COPY go.mod ./
# Agar go.sum nahi hai toh error na aaye isliye ise comment kar sakte hain 
# ya phir local terminal mein 'go mod tidy' chala kar upload karein.
RUN go mod download

# Sari files copy karna
COPY . .

# Build karna
RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
