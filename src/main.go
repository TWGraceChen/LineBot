package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	pb "linebot/service_client"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type config struct {
	servicePort        int
	serviceGrpc        int
	servicePath        string
	channelSecret      string
	channelAccessToken string
	dbHost             string
	dbPort             int
	dbUser             string
	dbPassword         string
	dbDbname           string
}

type userMode string

const (
	UserModeDefault  userMode = "尚未選擇任何功能"
	UserModeSong     userMode = "切換至歌曲管理功能"
	UserModeList     userMode = "切換至歌單管理功能"
	UserModeTemplate userMode = "切換至模板管理功能"
)

var (
	c       config
	bot     *linebot.Client
	gc      *grpc.ClientConn
	pycmd   *exec.Cmd
	sigchan chan bool
	users   map[string]userMode
	tmpSong map[string]songInfo
)

type songInfo struct {
	id    int
	name  string
	lyric string
}

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

func templateModeMessage(userid string) (reply []linebot.SendingMessage) {
	reply = []linebot.SendingMessage{linebot.NewTextMessage("已切換至模板模式，上傳一個pptx檔案即可建立模板。")}
	if db, err := connect(); err != nil {
		log.Println(err)
	} else {
		defer db.Close()
		sqlstat := fmt.Sprintf("select original_name,name,updatetime from template where userid = '%v' order by updatetime desc", userid)
		if rows, err := db.Query(sqlstat); err != nil {
			log.Println("query template info error:", err)
		} else {
			var actions []linebot.TemplateAction
			for rows.Next() {
				var original_name, name string
				var updatetime time.Time
				err = rows.Scan(&original_name, &name, &updatetime)
				if err != nil {
					log.Println("scan rows error:", err)
				} else {
					actions = append(actions, linebot.NewMessageAction(original_name, fmt.Sprintf("模板：%v,建立於：%v,系統檔案名稱：%v", original_name, updatetime.Format("2006-01-02 15:04:05"), name)))
				}

			}
			if len(actions) > 0 {
				reply = []linebot.SendingMessage{linebot.NewTemplateMessage("已切換至模板模式",
					linebot.NewButtonsTemplate("", "模板模式", "上傳一個pptx檔案即可建立模板，以下是已經建立的模板，可以點選後選擇下載或刪除。", actions...))}
			}
		}
	}
	return
}

func templateModeAction(userid string, m linebot.Message) (reply []linebot.SendingMessage) {
	reply = templateModeMessage(userid)
	switch message := m.(type) {
	case *linebot.FileMessage:
		content, err := bot.GetMessageContent(message.ID).Do()
		if err != nil {
			log.Println("get message content error:", err)
		} else {
			// check upload file is ppt file
			if !strings.HasSuffix(message.FileName, ".pptx") {
				reply = []linebot.SendingMessage{linebot.NewTextMessage("新增失敗，檔案非pptx檔。")}
				return
			}

			// save file
			newname := fmt.Sprintf("%v.pptx", time.Now().Unix())
			path := fmt.Sprintf("%v/template", c.servicePath)
			name := fmt.Sprintf("%v/%v", path, newname)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				if err := os.MkdirAll(path, 0755); err != nil {
					log.Println("create dir err:", err)
					reply = []linebot.SendingMessage{linebot.NewTextMessage("系統錯誤，上傳失敗")}
					return
				}
			}
			f, err := os.Create(name)
			if err != nil {
				log.Println("create file err:", err)
				reply = []linebot.SendingMessage{linebot.NewTextMessage("系統錯誤，上傳失敗")}
				return
			}
			defer f.Close()
			_, err = io.Copy(f, content.Content)
			if err != nil {
				log.Println("copy content to file err:", err)
				reply = []linebot.SendingMessage{linebot.NewTextMessage("系統錯誤，上傳失敗")}
				return
			}

			// save record
			if db, err := connect(); err != nil {
				log.Println(err)
				reply = []linebot.SendingMessage{linebot.NewTextMessage("系統錯誤，上傳失敗")}
				return
			} else {
				defer db.Close()
				insertStat := fmt.Sprintf("insert into template values ('%s','%s','%s',now())", userid, message.FileName, newname)
				if _, err := db.Exec(insertStat); err != nil {
					log.Println("save template log fail:", err)
					reply = []linebot.SendingMessage{linebot.NewTextMessage("系統錯誤，上傳失敗")}
					return
				}
			}

		}
		reply = []linebot.SendingMessage{linebot.NewTextMessage("模板已上傳，模板:" + message.FileName)}
		reply = append(reply, templateModeMessage(userid)...)
	case *linebot.TextMessage:
		t := message.Text
		r, _ := regexp.Compile("^(|下載|刪除)模板：(.*),建立於：(.*),系統檔案名稱：(.*)$")
		match := r.FindStringSubmatch(t)
		if len(match) == 0 {
			reply = templateModeMessage(userid)
			return
		}
		action := match[1]
		original_name := match[2]
		updatetime := match[3]
		name := match[4]
		if action == "" { // provide delete and download option
			reply = []linebot.SendingMessage{linebot.NewTemplateMessage("刪除/下載模板",
				linebot.NewConfirmTemplate(fmt.Sprintf("請選擇要刪除/下載模板：%v(建立於：%v)", original_name, updatetime),
					linebot.NewMessageAction("刪除", fmt.Sprintf("刪除模板：%v,建立於：%v,系統檔案名稱：%v", original_name, updatetime, name)),
					linebot.NewMessageAction("下載", fmt.Sprintf("下載模板：%v,建立於：%v,系統檔案名稱：%v", original_name, updatetime, name))))}
		} else if action == "下載" { // download template
			reply = []linebot.SendingMessage{linebot.NewTextMessage("尚未開放下載功能")}
		} else if action == "刪除" { // delete template
			// delete data
			if db, err := connect(); err != nil {
				log.Println(err)
			} else {
				defer db.Close()
				sqlstat := fmt.Sprintf("delete from template where userid = '%v' and name = '%v'", userid, name)
				if _, err := db.Exec(sqlstat); err != nil {
					log.Println("delete template record error:", err)
				}
			}

			// remove file
			path := fmt.Sprintf("%v/template/%v", c.servicePath, name)
			if err := os.Remove(path); err != nil {
				log.Println("remove file error:", err)
				reply = []linebot.SendingMessage{linebot.NewTextMessage("無法刪除檔案")}
				return
			}
			reply = []linebot.SendingMessage{linebot.NewTextMessage("模板刪除成功")}
		}

	}
	return
}

func songModeMessage(userid string) (reply []linebot.SendingMessage) {
	reply = []linebot.SendingMessage{linebot.NewTextMessage("已切換至歌曲模式，請輸入查詢或修改的歌名。")}
	return
}

func songModeAction(userid string, m linebot.Message) (reply []linebot.SendingMessage) {
	reply = songModeMessage(userid)
	switch message := m.(type) {
	case *linebot.TextMessage:
		t := message.Text
		r, _ := regexp.Compile("^(確認新增|確認修改|新增|修改)歌曲:(.+?)(,ID:(.+)){0,1}$")
		match := r.FindStringSubmatch(t)
		tmpSongInfo, ok := tmpSong[userid]
		if len(match) == 0 && len(t) < 20 {
			// search
			if db, err := connect(); err != nil {
				log.Println(err)
			} else {
				defer db.Close()
				var id int
				var displayname, lyric string
				sqlstat := fmt.Sprintf("select id,displayname,content from lyrics where displayname = '%v'", t)
				if err := db.QueryRow(sqlstat).Scan(&id, &displayname, &lyric); err != nil {
					// not exist
					reply = []linebot.SendingMessage{linebot.NewTemplateMessage(t,
						linebot.NewButtonsTemplate("", t, "是否要新增歌詞",
							linebot.NewMessageAction("新增", fmt.Sprintf("新增歌曲:%v", t))))}
				} else {
					// exist
					reply = []linebot.SendingMessage{linebot.NewTextMessage(lyric)}
					reply = append(reply, linebot.NewTemplateMessage(displayname,
						linebot.NewButtonsTemplate("", displayname, "歌詞在上面",
							linebot.NewMessageAction("修改", fmt.Sprintf("修改歌曲:%v,ID:%v", displayname, id)))))
				}
			}
		} else if len(match) > 0 && (match[1] == "新增" || match[1] == "修改") {
			name := match[2]
			var tmp songInfo
			tmp.name = name
			if match[1] == "新增" {
				tmp.id = -1
			} else if match[1] == "修改" {
				id, _ := strconv.Atoi(match[4])
				tmp.id = id
			}
			tmpSong[userid] = tmp
			reply = []linebot.SendingMessage{linebot.NewTextMessage(fmt.Sprintf("請輸入「%v」的歌詞", name))}
		} else if len(match) > 0 && (match[1] == "確認新增" || match[1] == "確認修改") && ok {
			name := match[2]
			if name != tmpSongInfo.name {
				reply = []linebot.SendingMessage{linebot.NewTextMessage("操作錯誤")}
				return
			}
			if match[1] == "確認新增" {
				if db, err := connect(); err != nil {
					log.Println(err)
				} else {
					defer db.Close()
					var maxid int
					sqlstat := "select max(id) from lyrics"
					if err := db.QueryRow(sqlstat).Scan(&maxid); err != nil {
						log.Println(err)
					}
					sqlstat = fmt.Sprintf("insert into lyrics values (%v,'%v','%v','%v',now())", maxid+1, name, name, tmpSongInfo.lyric)
					if _, err := db.Exec(sqlstat); err != nil {
						log.Println(err)
					}
					reply = []linebot.SendingMessage{linebot.NewTextMessage(fmt.Sprintf("「%v」新增成功", name))}
					return
				}
			} else if match[1] == "確認修改" {
				id, _ := strconv.Atoi(match[4])
				if id != tmpSongInfo.id {
					reply = []linebot.SendingMessage{linebot.NewTextMessage("操作錯誤")}
					return
				}
				if db, err := connect(); err != nil {
					log.Println(err)
				} else {
					defer db.Close()
					sqlstat := fmt.Sprintf("update lyrics set content = '%v',updatetime = now() where id = %v ", tmpSongInfo.lyric, tmpSongInfo.id)
					if _, err := db.Exec(sqlstat); err != nil {
						log.Println(err)
					}
					reply = []linebot.SendingMessage{linebot.NewTextMessage(fmt.Sprintf("「%v」新增成功", name))}
					return
				}
			}
			delete(tmpSong, userid)
		} else if ok {
			tmpSongInfo.lyric = t
			tmpSong[userid] = tmpSongInfo
			if tmpSongInfo.id == -1 {
				reply = []linebot.SendingMessage{linebot.NewTemplateMessage(tmpSongInfo.name,
					linebot.NewButtonsTemplate("", tmpSongInfo.name, "是否確認新增",
						linebot.NewMessageAction("確認", fmt.Sprintf("確認新增歌曲:%v", tmpSongInfo.name))))}
			} else {
				reply = []linebot.SendingMessage{linebot.NewTemplateMessage(tmpSongInfo.name,
					linebot.NewButtonsTemplate("", tmpSongInfo.name, "是否確認修改",
						linebot.NewMessageAction("確認", fmt.Sprintf("確認修改歌曲:%v,ID:%v", tmpSongInfo.name, tmpSongInfo.id))))}
			}
		}

	}
	return
}

func listModeMessage(userid string) (reply []linebot.SendingMessage) {
	reply = []linebot.SendingMessage{linebot.NewTextMessage("已切換至歌單模式，請用以下的格式輸入歌單：模板：<模板名稱>\n<詩歌1名稱>\n<詩歌2名稱>\n<詩歌3名稱>\n經文:<經文出處>")}
	return
}
func listModeAction(userid string, m linebot.Message) (reply []linebot.SendingMessage) {
	return
}

func processMessage(userid, replytoken string, m linebot.Message) (reply []linebot.SendingMessage) {
	if db, err := connect(); err != nil {
		log.Println(err)
	} else {
		defer db.Close()
		insertStat := fmt.Sprintf("insert into linelog values ('%s','%s','%s',now())", userid, m.Type(), replytoken)
		if _, err := db.Exec(insertStat); err != nil {
			log.Println("save user profile fail:", err)
		}
	}

	// prepare default message
	defaultReply := linebot.NewImagemapMessage("https://github.com/TWGraceChen/LineBot/blob/main/src/img/modeimg.jpg?raw=true",
		"請輸入要切換的模式",
		linebot.ImagemapBaseSize{Width: 1040, Height: 1040},
		linebot.NewMessageImagemapAction("歌曲管理", string(UserModeSong), linebot.ImagemapArea{X: 0, Y: 0, Width: 1040, Height: 520}),
		linebot.NewMessageImagemapAction("歌單管理", string(UserModeList), linebot.ImagemapArea{X: 0, Y: 520, Width: 520, Height: 520}),
		linebot.NewMessageImagemapAction("模板管理", string(UserModeTemplate), linebot.ImagemapArea{X: 520, Y: 520, Width: 520, Height: 520}))

	// check message is switch message
	switch message := m.(type) {
	case *linebot.TextMessage:
		switch message.Text {
		case string(UserModeList): // switch to List mode
			users[userid] = UserModeList
			reply = listModeMessage(userid)
			return
		case string(UserModeSong): // switch to Song mode
			users[userid] = UserModeSong
			reply = songModeMessage(userid)
			return
		case string(UserModeTemplate): // switch to Template mode
			users[userid] = UserModeTemplate
			reply = templateModeMessage(userid)
			return
		case "0": // switch to default mode
			users[userid] = UserModeDefault
		}
	}

	switch users[userid] {
	case UserModeTemplate:
		reply = templateModeAction(userid, m)
	case UserModeList:
		reply = listModeAction(userid, m)
	case UserModeSong:
		reply = songModeAction(userid, m)
	default:
		reply = append(reply, defaultReply)
	}
	return

}

func processEvent(events []*linebot.Event) {
	for _, event := range events {
		// read and save user profile
		userid, displayname := getUserProfile(event)
		if db, err := connect(); err != nil {
			log.Println(err)
		} else {
			defer db.Close()
			insertStat := fmt.Sprintf("insert into lineuser values ('%s','%s',now()) on conflict (userid) do nothing;", userid, displayname)
			if _, err := db.Exec(insertStat); err != nil {
				log.Println("save user profile fail:", err)
			}
		}

		// check user in users map
		if _, ok := users[userid]; !ok {
			users[userid] = UserModeDefault
		}

		// switch type of event(message,follow,join...)
		switch event.Type {
		case linebot.EventTypeMessage:
			reply := processMessage(userid, event.ReplyToken, event.Message)
			if _, err := bot.ReplyMessage(event.ReplyToken, reply...).Do(); err != nil {
				log.Print(err)
			}
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
	c.servicePath = v.GetString("service.path")
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

	sqlstat = `create table if not exists template (userid varchar(64),original_name varchar(64),name varchar(64),updatetime timestamp)`
	if _, err := db.Exec(sqlstat); err != nil {
		return err
	}

	sqlstat = `create table if not exists lyrics (id bigint,name varchar(64),displayname varchar(64),content text,updatetime timestamp)`
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

	// init users mode map
	users = make(map[string]userMode)
	tmpSong = make(map[string]songInfo)

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
