# LineBot

## Information Flow
```mermaid
flowchart 
    subgraph Infomation_Flow
    direction LR
    START{{start}} --> U
    U((user)) -- 發送訊息 --> L["Line<br/>Platform"]
    L -- 傳遞訊息 --> G["LineBot<br/>service<br/>(Golang)"]
    G -- 發送訊息 --> L
    L -- 傳遞訊息 --> U
        subgraph Server
            G["LineBot<br/>service<br/>(Golang)"] -- gRPC --> P["Customized<br/>Service<br/>(Python)"]
            G --> D[("DB<br/>(Postgres)")] 
        end
    end
    classDef bg fill:#fff,stroke-width:0px;
    class Infomation_Flow bg

```

## Flow Chart
```mermaid
sequenceDiagram
    actor User
    critical default mode
    User->>+LineBot: Message Event
    LineBot-->>-User: 功能選單
    User->>+LineBot: 管理模板/管理歌曲/建立歌單
    LineBot->>LineBot: 切換user模式
    LineBot-->>-User: 已切換模式
    
    option 管理模板模式
    User ->>+ LineBot: File Message
    LineBot ->> LineBot: 儲存檔案
    LineBot ->>+ DB: save log
    DB -->- LineBot: 儲存成功
    LineBot -->>- User: 模板上傳成功
    User ->>+ LineBot: Message:模板名稱
    LineBot -->>- User: 提供下載&刪除選單
    User ->>+ LineBot: 下載模板
    LineBot -->>- User: 回傳檔案
    User ->>+ LineBot: 刪除模板
    LineBot ->>+ DB: delete log
    DB -->- LineBot: 刪除成功
    LineBot ->> LineBot: 刪除檔案
    LineBot -->>- User: 模板刪除成功
    User ->>+ LineBot: "0"
    LineBot ->> LineBot: 切換user模式:default
    LineBot -->>- User: 功能選單
    
    option 管理歌曲模式
    User ->>+ LineBot: message:歌曲名稱
    LineBot ->>+ DB: 查詢歌曲
    critical
    DB -->>- LineBot: 查詢結果
    option 有結果
    LineBot -->>- User: 歌詞 & 是否要修改選項
    User ->>+ LineBot: 修改歌詞
    LineBot ->> LineBot: 儲存變數
    LineBot -->> User: 請輸入修改歌詞
    User ->> LineBot: 修改的歌詞
    LineBot -->> User: 預覽及確認選項
    User ->> LineBot: 確認修改
    LineBot ->+ DB: 儲存
    DB -->>- LineBot: 儲存成功
    LineBot -->>- User: 修改成功
    
    option 沒結果
    LineBot -->> User: 是否要新增選項
    User ->>+ LineBot: 新增歌詞
    LineBot ->> LineBot: 儲存變數
    LineBot -->> User: 請輸入新增歌詞
    User ->> LineBot: 新增的歌詞
    LineBot -->> User: 預覽及確認選項
    User ->> LineBot: 確認新增
    LineBot ->+ DB: 儲存
    DB -->>- LineBot: 儲存成功
    LineBot -->>- User: 新增成功
    end
   
    
    User ->>+ LineBot: "0"
    LineBot ->> LineBot: 切換user模式:default
    LineBot -->- User: 功能選單
    END

```

## 建立與佈署Line Bot
1. 在Line Developers建立Provider和Channel
https://developers.line.biz/en/
2. 完成基本設置，取得channel_token和channel_secret
參考：https://ithelp.ithome.com.tw/articles/10229943
3. 安裝server環境：`./install.sh`(未完成)  
4. 編譯程式碼：`./build.sh`  
5. 複製config.yaml.sample: `cp config.yaml.sample config.yaml`
6. [編輯config.yaml](#編輯configyaml)
7. 啟動db: `docker-compose up -d`
8. 啟動service：`./linebot`
9. 使用ngrok將local的port forword出去(如果允許最好用更安全的方法)  
-- 安裝ngrok：https://ngrok.com/download  
-- 註冊登入後，取得token：https://dashboard.ngrok.com/get-started/your-authtoken  
-- 在local新增token：`ngrok config add-authtoken <AUTH_TOKEN>`  
-- 啟動ngrok client：`ngrok http <PORT>`  
-- 取得URL  
10. 回到Line Developers填入webhook URL
11. 完成


## 編輯config.yaml
### service
- port: LineBot service的http埠號  
- grpc: python grpc server的埠號  
- path: 用於存取資料的路徑  

### linebot
- channel_secret: 從line console取得的channel secret  
- channel_access_token: 從line console取得的channel access token  

### db
- host,port,user,password,database: db的連線資訊  

---

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
