package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/k0kubun/pp"
	log "github.com/sirupsen/logrus"
	yarb "github.com/wmw9/yarb-struct"
	tb "gopkg.in/tucnak/telebot.v2"
	"io"
	"net/http"
	"strings"
	"time"
)

var tg *tb.Bot

func handlePing(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func handlePost(c *gin.Context) {
	// Unmarshal payload into struct
	var p yarb.Payload
	if err := c.ShouldBindJSON(&p); err != nil {
		log.Println("Couldn't bind json")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pp.Println(p)

	if err := sendToTelegram(p); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func sendToTelegram(p yarb.Payload) error {
	// Initialize bot
	tg, err := tb.NewBot(tb.Settings{
		Token:  TelegramToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalln("Telegram", err)
		return err
	}

	// Prepare files
	album := prepareFilesReader(p.Files)
	pp.Println(album)

	// Send post to telegram
	_, err = tg.SendAlbum(tb.ChatID(p.TelegramChanID), album)

	if err != nil {
		log.Println("Send video", err, album)
		return err
	}
	return nil
}

func prepareFiles(files []string) tb.Album {
	var album tb.Album

	for _, v := range files {
		if strings.Contains(v, ".jpg?") {
			println("contains .jpg:", v)
			album = append(album, &tb.Photo{File: tb.FromURL(v)})
		}
		if strings.Contains(v, ".mp4?") {
			println("contains .mp4:", v)
			album = append(album, &tb.Video{File: tb.FromURL(v)})
		}
	}

	return album
}

func prepareFilesReader(files []string) tb.Album {
	var album tb.Album

	for _, v := range files {
		if strings.Contains(v, ".jpg") {
			println("contains .jpg:", v)
			data := get(v)
			album = append(album, &tb.Photo{File: tb.FromReader(data)})
		}
		if strings.Contains(v, ".mp4") {
			println("contains .mp4:", v)
			data := get(v)
			album = append(album, &tb.Video{File: tb.FromReader(data)})
		}
	}

	return album
}

func UpdateIGStoriesTs(p yarb.Payload) error {
	url := fmt.Sprintf("http://%v/yarb/user/name/%v/date/instagram_stories/%v", YarbDBApiURL, p.Person, p.Timestamp)
	log.Debugf("%v\n", url)
	client := resty.New()
	client.SetBasicAuth(yarbBasicAuthUser, yarbBasicAuthPass)
	resp, err := client.R().Get(url)
	if err != nil {
		return err
	}
	log.Infof("%v:\n%v", resp.String())
	return nil
}

func get(v string) io.Reader {
	println("get")
	resp, err := http.Get(v)
	if err != nil {
		log.Println("Failed to fetch data from HTTP URL")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to io.ReadAll: %v", err)
	}
	return bytes.NewReader(data)
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	authorized := r.Group("/yarb/telegram", gin.BasicAuth(gin.Accounts{
		yarbBasicAuthUser: yarbBasicAuthPass,
	}))

	authorized.GET("/ping", handlePing)
	authorized.POST("/post", handlePost)

	return r
}
