FROM golang:1.22.4-bullseye AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod tidy && go mod download

COPY  . .
RUN make build


FROM golang:1.22.4-bullseye

WORKDIR /apps

COPY --from=builder /workspace/build/storage-bench /usr/bin/storage-bench

CMD ["storage-bench"]
