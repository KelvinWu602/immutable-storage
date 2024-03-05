FROM golang:1.20

WORKDIR /immutable-storage
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o /usr/local/bin/immutable-storage

CMD ["immutable-storage"]