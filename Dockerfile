FROM golang:1.14.6 AS build-env

ENV GO111MODULE=on

WORKDIR /app/postgresql-backup
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN go build -o postgresql-backup
RUN chmod +x postgresql-backup

CMD ["./postgresql-backup"]