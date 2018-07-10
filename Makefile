proto:
	cd internal/proto && protoc --go_out=plugins=grpc:. upload_service.proto

server:
	go build -o bin/uploader github.com/Code-Hex/upload/app/uploader

client:
	go build -o bin/client github.com/Code-Hex/upload/cmd/client

credential:
	openssl genrsa -des3 2048 -out ./bin/tls/server.key
	openssl req -new -key ./bin/tls/server.key -sha256 -out ./bin/tls/server.csr
	openssl req -days 365 -in ./bin/tls/server.csr -key ./bin/tls/server.key -x509 -out ./bin/tls/server.crt

.PHONY: proto server credential client