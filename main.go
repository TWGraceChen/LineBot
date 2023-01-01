package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"

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
						reply = linebot.NewTextMessage("text message $").AddEmoji(linebot.NewEmoji(13, "5ac21e6c040ab15980c9b444", "001"))
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
					case "7":
						reply = linebot.NewImagemapMessage("https://thumbs.dreamstime.com/b/seascape-over-under-sea-cloudy-sky-rocky-seabed-seascape-over-under-sea-surface-cloudy-blue-sky-rocky-seabed-underwater-108661258.jpg",
							"this is alt infomation of imagemap.",
							linebot.ImagemapBaseSize{Width: 1040, Height: 1040},
							linebot.NewURIImagemapAction("Left up label", "https://www.facebook.com/", linebot.ImagemapArea{X: 0, Y: 0, Width: 520, Height: 520}),
							linebot.NewURIImagemapAction("Right up label", "https://www.google.com/", linebot.ImagemapArea{X: 520, Y: 0, Width: 520, Height: 520}),
							linebot.NewMessageImagemapAction("left down label", "tap left.", linebot.ImagemapArea{X: 0, Y: 520, Width: 520, Height: 520}),
							linebot.NewMessageImagemapAction("right down label", "tap right.", linebot.ImagemapArea{X: 520, Y: 520, Width: 520, Height: 520}))

					case "8":
						reply = linebot.NewTemplateMessage("Buttons Template.",
							linebot.NewButtonsTemplate("https://www.tastingtable.com/img/gallery/coffee-brands-ranked-from-worst-to-best/l-intro-1645231221.jpg", "Title", "this is text field",
								linebot.NewURIAction("URI Action", "https://www.google.com/"),
								linebot.NewMessageAction("Message Action", "This is a message action.")))
					case "9":
						reply = linebot.NewTemplateMessage("Confirm Template.",
							linebot.NewConfirmTemplate("Confirm Template",
								linebot.NewURIAction("left", "https://www.google.com/"),
								linebot.NewMessageAction("right", "This is a message action.")))

					case "10":
						reply = linebot.NewTemplateMessage("Carousel Template.",
							linebot.NewCarouselTemplate(
								linebot.NewCarouselColumn("https://www.tastingtable.com/img/gallery/coffee-brands-ranked-from-worst-to-best/l-intro-1645231221.jpg", "Title", "this is text field",
									linebot.NewURIAction("URI Action", "https://www.google.com/"),
									linebot.NewMessageAction("Message Action", "This is a message action.")),
								linebot.NewCarouselColumn("https://www.tastingtable.com/img/gallery/coffee-brands-ranked-from-worst-to-best/l-intro-1645231221.jpg", "Title", "this is text field",
									linebot.NewURIAction("URI Action", "https://www.google.com/"),
									linebot.NewMessageAction("Message Action", "This is a message action."))))

					case "11":
						reply = linebot.NewTemplateMessage("Image carousel Template.",
							linebot.NewImageCarouselTemplate(
								linebot.NewImageCarouselColumn("https://www.tastingtable.com/img/gallery/coffee-brands-ranked-from-worst-to-best/l-intro-1645231221.jpg",
									linebot.NewURIAction("URI Action", "https://www.google.com/")),
								linebot.NewImageCarouselColumn("https://www.tastingtable.com/img/gallery/coffee-brands-ranked-from-worst-to-best/l-intro-1645231221.jpg",
									linebot.NewURIAction("URI Action", "https://www.google.com/"))))
					default:
						reply = linebot.NewTextMessage("1:Text\n2:Sticker\n3:Image\n4:Video\n5:Audio\n6:Location\n7:Imagemap\n8:Buttun Template\n9:Confirm Tempplate\n10:Carousel Template\n11:Image carousel Template")
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
					content, err := bot.GetMessageContent(message.ID).Do()
					if err != nil {
						log.Print(err)
					}
					b, err := io.ReadAll(content.Content)
					if err != nil {
						log.Print(err)
					} else {
						img, _, err := image.Decode(bytes.NewReader(b))
						if err != nil {
							log.Fatalln(err)
						}

						out, _ := os.Create("./" + message.ID + ".jpeg")
						defer out.Close()

						var opts jpeg.Options
						opts.Quality = 100

						err = jpeg.Encode(out, img, &opts)
						if err != nil {
							log.Println(err)
						}
					}
					replyMessage := fmt.Sprintf("image ID is %s, length is %v,type is %v", message.ID, content.ContentLength, content.ContentType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.AudioMessage:
					content, err := bot.GetMessageContent(message.ID).Do()
					if err != nil {
						log.Print(err)
					}

					// TODO: save audio file

					replyMessage := fmt.Sprintf("audio ID is %s, length is %v,type is %v", message.ID, content.ContentLength, content.ContentType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}

				case *linebot.VideoMessage:
					content, err := bot.GetMessageContent(message.ID).Do()
					if err != nil {
						log.Print(err)
					}

					// TODO: save video file

					replyMessage := fmt.Sprintf("video ID is %s, dur is %v, length is %v,type is %v", message.ID, message.Duration, content.ContentLength, content.ContentType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}

				case *linebot.FileMessage:
					content, err := bot.GetMessageContent(message.ID).Do()
					if err != nil {
						log.Print(err)
					}

					// TODO: save file

					replyMessage := fmt.Sprintf("file ID is %s, filename is %v, length is %v,type is %v", message.ID, message.FileName, content.ContentLength, content.ContentType)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do(); err != nil {
						log.Print(err)
					}
				case *linebot.LocationMessage:
					replyMessage := fmt.Sprintf("title is %s, address is %s, lat is %v, lon is %v", message.Title, message.Address, message.Latitude, message.Longitude)
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
