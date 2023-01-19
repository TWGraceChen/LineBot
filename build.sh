python3 -m grpc_tools.protoc -I ./src --python_out=./src/service --pyi_out=./src/service --grpc_python_out=./src/service service.proto
export PATH="$PATH:$(go env GOPATH)/bin"
protoc --go_out=./src/service_client --go_opt=paths=source_relative --go-grpc_out=./src/service_client --go-grpc_opt=paths=source_relative ./src/service.proto
cd src;go build -o ../linebot