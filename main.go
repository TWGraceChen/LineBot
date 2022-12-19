package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/spf13/viper"
)

var (
	port = flag.Int("p", 5000, "port number.")
	bot  *linebot.Client
)

func getUserProfile(e *linebot.Event) {
	switch e.Source.Type {
	case linebot.EventSourceTypeUser:
		log.Println("from user:" + e.Source.UserID)
		userProfile, err := bot.GetProfile(e.Source.UserID).Do()
		if err != nil {
			log.Println("Get user profile fail:", err)
		} else {
			log.Println("Get user profile success!!")
			log.Println("UserID:" + userProfile.UserID)
			log.Println("DisplayName:" + userProfile.DisplayName)
			log.Println("PictureURL:" + userProfile.PictureURL)
			log.Println("StatusMessage:" + userProfile.StatusMessage)
			log.Println("Language:" + userProfile.Language)
		}
	case linebot.EventSourceTypeGroup:
		log.Println("from group:" + e.Source.GroupID)
	case linebot.EventSourceTypeRoom:
		log.Println("from room:" + e.Source.RoomID)
	}
}

func main() {
	flag.Parse()

	// read config file
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()
	if err != nil {
		log.Fatal("[Error] Loading config failed: ", err)
	}
	channelSecret := v.GetString("linebot.channel_secret")
	channelAccessToken := v.GetString("linebot.channel_access_token")

	// create a linebot
	bot, err = linebot.New(channelSecret, channelAccessToken)
	if err != nil {
		log.Fatal(err)
	}

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {

				getUserProfile(event)

				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.StickerMessage:
					replyMessage := fmt.Sprintf(
						"sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	})
	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	url := fmt.Sprintf(":%v", *port)
	log.Println("listing:" + url)
	if err := http.ListenAndServe(url, nil); err != nil {
		log.Fatal(err)
	}
}
