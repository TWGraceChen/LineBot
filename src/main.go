package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	pb "linebot/service_client"
	"log"
	"net/http"
	"os/exec"
	"time"

	_ "github.com/lib/pq"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type config struct {
	servicePort        int
	serviceGrpc        int
	channelSecret      string
	channelAccessToken string
	dbHost             string
	dbPort             int
	dbUser             string
	dbPassword         string
	dbDbname           string
}

var (
	c     config
	bot   *linebot.Client
	gc    *grpc.ClientConn
	pycmd *exec.Cmd
)

var (
	sigchan chan bool
)

func getUserProfile(e *linebot.Event) (userid, displayname string) {
	switch e.Source.Type {
	case linebot.EventSourceTypeUser:
		userProfile, err := bot.GetProfile(e.Source.UserID).Do()
		if err != nil {
			log.Println("Get user profile fail:", err)
		} else {
			//log.Printf("UserID:%v, DisplayName:%v, PictureURL:%v, StatusMessage:%v, Language:%v",
			//	userProfile.UserID, userProfile.DisplayName, userProfile.PictureURL, userProfile.StatusMessage, userProfile.Language)
			userid = userProfile.UserID
			displayname = userProfile.DisplayName
		}
	case linebot.EventSourceTypeGroup:
		log.Println("from group:" + e.Source.GroupID)
	case linebot.EventSourceTypeRoom:
		log.Println("from room:" + e.Source.RoomID)
	}
	return
}

func searchLyric(songname string) (lyric string) {
	c := pb.NewMyServiceClient(gc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SearchLyric(ctx, &pb.Searchinfo{Name: songname})
	if err != nil {
		log.Printf("could not search: %v", err)
	}
	return r.GetLyric()
}

func processMessage(userid, replytoken string, m linebot.Message) {
	if db, err := connect(); err != nil {
		log.Println(err)
	} else {
		defer db.Close()
		insertStat := fmt.Sprintf("insert into linelog values ('%s','%s','%s',now())", userid, m.Type(), replytoken)
		if _, err := db.Exec(insertStat); err != nil {
			log.Println("save user profile fail:", err)
		}
	}
	var reply linebot.SendingMessage
	switch message := m.(type) {
	case *linebot.TextMessage:
		lyric := searchLyric(message.Text)
		reply = linebot.NewTextMessage(lyric)
	default:
		log.Println("message type:", m.Type())
		reply = linebot.NewTextMessage("請輸入文字")
	}

	if _, err := bot.ReplyMessage(replytoken, reply).Do(); err != nil {
		log.Print(err)
	}
}

func processEvent(events []*linebot.Event) {
	for _, event := range events {
		// read and save user profile
		userid, displayname := getUserProfile(event)
		if db, err := connect(); err != nil {
			log.Println(err)
		} else {
			defer db.Close()
			insertStat := fmt.Sprintf("insert into lineuser values ('%s','%s',now())", userid, displayname)
			if _, err := db.Exec(insertStat); err != nil {
				log.Println("save user profile fail:", err)
			}
		}

		// switch type of event(message,follow,join...)
		switch event.Type {
		case linebot.EventTypeMessage:
			processMessage(userid, event.ReplyToken, event.Message)
		default:
			log.Println("event:", event.Type)
		}

	}
}

func readConfig() (c config, err error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err = v.ReadInConfig()
	if err != nil {
		return c, err

	}
	c.servicePort = v.GetInt("service.port")
	c.serviceGrpc = v.GetInt("service.grpc")
	c.channelSecret = v.GetString("linebot.channel_secret")
	c.channelAccessToken = v.GetString("linebot.channel_access_token")
	c.dbHost = v.GetString("db.host")
	c.dbPort = v.GetInt("db.port")
	c.dbUser = v.GetString("db.user")
	c.dbPassword = v.GetString("db.password")
	c.dbDbname = v.GetString("db.database")
	log.Println("Read config file Success.")
	return c, nil
}

func connect() (db *sql.DB, err error) {
	db, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=disable", c.dbHost, c.dbPort, c.dbUser, c.dbPassword, c.dbDbname))
	if err != nil {
		return db, err
	}
	return
}

func initdb() (err error) {
	db, err := connect()
	if err = db.Ping(); err != nil {
		return err
	}
	defer db.Close()
	sqlstat := `create table if not exists lineuser (userid varchar(64) primary key,displayname varchar(64),updatetime timestamp)`
	if _, err := db.Exec(sqlstat); err != nil {
		return err
	}

	sqlstat = `create table if not exists linelog (userid varchar(64),messagetype varchar(64),replytoken varchar(64),receicetime timestamp)`
	if _, err := db.Exec(sqlstat); err != nil {
		return err
	}

	sqlstat = `create table if not exists lyrics (name varchar(64),content text,updatetime timestamp)`
	if _, err := db.Exec(sqlstat); err != nil {
		return err
	}

	log.Println("Init Database Success.")
	return nil
}

func closeGrpc() {
	gc.Close()      // clost grpc connection
	sigchan <- true // deliver signal to kill python grpc server
}

func initGrpc() (err error) {
	// start python grpc server
	pycmd = exec.Command("python3", "src/service/server.py", fmt.Sprintf("%v", c.serviceGrpc))
	err = pycmd.Start()
	if err != nil {
		return err
	}

	// waiting for kill python grpc server process
	sigchan = make(chan bool)
	go func() {
		for {
			select {
			case <-sigchan:
				pycmd.Process.Kill()
			}
		}

	}()

	// establish connection to python grpc server
	addr := fmt.Sprintf("localhost:%v", c.serviceGrpc)
	gc, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return err
	}
	log.Println("Init gRPC Success.")
	return nil
}

func main() {
	flag.Parse()

	// read config file
	var err error
	c, err = readConfig()
	if err != nil {
		log.Fatal("[Error] Loading config failed: ", err)
	}

	// init database
	if err = initdb(); err != nil {
		log.Fatal("[Error] Init Database failed:", err)
	}

	// grpc connection
	err = initGrpc()
	if err != nil {
		log.Fatal("[Error] Init gRPC connection failed:", err)
	}
	defer closeGrpc()

	// create a linebot
	bot, err = linebot.New(c.channelSecret, c.channelAccessToken)
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
		processEvent(events)
	})

	// Active htttp service
	url := fmt.Sprintf(":%v", c.servicePort)
	log.Println("listening on:" + url)
	if err := http.ListenAndServe(url, nil); err != nil {
		log.Fatal(err)
	}
}
