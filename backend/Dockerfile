FROM golang:1.22-alpine

WORKDIR /app
COPY go.mod ./
COPY . ./
RUN go mod download
EXPOSE 8080
CMD ["go", "run", "./cmd/api"]
