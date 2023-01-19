# LineBot

## 建立與佈署Line Bot
1. 在Line Developers建立Provider和Channel
https://developers.line.biz/en/
2. 完成基本設置，取得channel_token和channel_secret
參考：https://ithelp.ithome.com.tw/articles/10229943
3. 啟動go http service
4. 使用ngrok將local的port forword出去  
-- 安裝ngrok：https://ngrok.com/download  
-- 註冊登入後，取得token：https://dashboard.ngrok.com/get-started/your-authtoken  
-- 在local新增token：`ngrok config add-authtoken <AUTH_TOKEN>`  
-- 啟動ngrok client：`ngrok http <PORT>`  
-- 取得URL  
5. 回到Line Developers填入webhook URL
6. 完成

## gRPC
### python
- Install: `python3 -m pip install grpcio`
- install gRPC tools: `python3 -m pip install grpcio-tools`
- Generate gRPC code: `python3 -m grpc_tools.protoc -I ./ --python_out=./service --pyi_out=./service --grpc_python_out=./service service.proto`
- start server: `python3 server.py`


### golang
- `go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28`
- `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2`
- `export PATH="$PATH:$(go env GOPATH)/bin"`
- Generate gRPC code : `protoc --go_out=./service_client --go_opt=paths=source_relative --go-grpc_out=./service_client --go-grpc_opt=paths=source_relative service.proto`