BINARY_NAME = storage-bench

build:
	go build -o build/${BINARY_NAME} main.go


docker-build:
	docker build -t kevin2025/${BINARY_NAME} .

docker-run: docker-build
	docker run -d kevin2025/${BINARY_NAME}