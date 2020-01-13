FROM golang:1.13-alpine
LABEL MAINTAINER="Dinesh Katwal<dinesh@auzmor.com>"
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o notification-service .
EXPOSE 8080
ENTRYPOINT [ "./notification-service" ]