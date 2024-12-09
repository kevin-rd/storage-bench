FROM golang:1.22.4-bullseye AS builder

WORKDIR /go/cache
COPY go.mod go.sum ./
RUN go mod download

WORKDIR /workspace
COPY  . .
RUN go build -o build/storage-bench main.go


FROM golang:1.22.4-bullseye

WORKDIR /apps

COPY --from=builder /workspace/build/storage-bench /usr/bin/storage-bench

CMD ["storage-bench"]
