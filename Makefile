.PHONY: gen-proto build-all run-dht run-tracker run-web test tidy fmt vet

# proto 生成：module 根在 api/proto，输出到 api/gen
gen-proto:
	cd api/proto && buf generate

build-all:
	mkdir -p bin
	go build -o ./bin/dht-service ./services/dht-service/cmd
	go build -o ./bin/tracker-service ./services/tracker-service/cmd
	go build -o ./bin/web-server ./services/web-server/cmd

run-dht:
	go run ./services/dht-service/cmd

run-tracker:
	go run ./services/tracker-service/cmd

run-web:
	go run ./services/web-server/cmd

tidy:
	go mod tidy

fmt:
	gofmt -w ./pkg ./services ./internal

vet:
	go vet ./...

test:
	go test -v ./...
