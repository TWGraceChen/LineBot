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

## TO DO
- 把token和secret拉到外部config [OK]
- port用flag來指定 [OK]
- reply各種type的message[OK]
- get各種type的message[OK]
- push message
- event type
- logging
- 把data存到db
- 啟動python的gRPC server，讓go可以使用python的function。
- dockerize
