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
		userProfile, err := bot.GetProfile(e.Source.UserID).Do()
		if err != nil {
			log.Println("Get user profile fail:", err)
		} else {
			log.Printf("UserID:%v, DisplayName:%v, PictureURL:%v, StatusMessage:%v, Language:%v",
				userProfile.UserID, userProfile.DisplayName, userProfile.PictureURL, userProfile.StatusMessage, userProfile.Language)
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
				// read user profile
				getUserProfile(event)

				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					// reply different type of message base on request message content
					var reply linebot.SendingMessage
					switch message.Text {
					case "1":
						reply = linebot.NewTextMessage("text message")
					case "2":
						reply = linebot.NewStickerMessage("446", "1988")
					case "3":
						reply = linebot.NewImageMessage("https://images.pexels.com/photos/1001682/pexels-photo-1001682.jpeg?cs=srgb&dl=pexels-kellie-churchman-1001682.jpg&fm=jpg", "https://images.pexels.com/photos/1001682/pexels-photo-1001682.jpeg?cs=srgb&dl=pexels-kellie-churchman-1001682.jpg&fm=jpg")
					case "4":
						reply = linebot.NewVideoMessage("https://file-examples.com/wp-content/uploads/2017/04/file_example_MP4_480_1_5MG.mp4", "https://file-examples.com/wp-content/uploads/2017/04/file_example_MP4_480_1_5MG.mp4")
					case "5":
						reply = linebot.NewAudioMessage("https://file-examples.com/wp-content/uploads/2017/11/file_example_MP3_700KB.mp3", 10)
					case "6":
						reply = linebot.NewLocationMessage("My location", "台北市中正區忠孝東路一段9號4樓", 25.0450205, 121.5214747)
					default:
						reply = linebot.NewTextMessage("1:Text\n2:Sticker\n3:Image\n4:Video\n5:Audio\n6:Location")
					}

					if _, err = bot.ReplyMessage(event.ReplyToken, reply).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.StickerMessage:
					replyMessage := fmt.Sprintf(
						"sticker id is %s, stickerResourceType is %s", message.StickerID, message.StickerResourceType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.ImageMessage:
					replyMessage := fmt.Sprintf("image ID is %s", message.ID)
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
