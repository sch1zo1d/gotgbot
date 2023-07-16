package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
	youtube "github.com/kkdai/youtube/v2"
)

var (
	messageid int
	url_video string
	is_audio  bool
	// chatcallbackId int

	// is_continue    bool

	viauKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Видео", "Видео"),
			tgbotapi.NewInlineKeyboardButtonData("Аудио", "Аудио"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад", "Назад"),
		))

	audioKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Продолжить в telegram", "Продолжить в telegram")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Скачать через сайт (в разработке)", "Скачать через сайт")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Изменить название файла (в разработке)", "Изменить название файла")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад", "Назад")))

	videoKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Продолжить в telegram", "Продолжить в telegram")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Скачать через сайт (в разработке)", "Скачать через сайт")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Изменить качество видео (по умолчанию - лучшее) (в разработке)", "Изменить качество видео")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад", "Назад")))
)

type (
	VideoConfig     = tgbotapi.VideoConfig
	AudioConfig     = tgbotapi.AudioConfig
	BaseFile        = tgbotapi.BaseFile
	RequestFileData = tgbotapi.RequestFileData
	BaseChat        = tgbotapi.BaseChat
)

func init_env() {
	envfile, err := os.Open(".env")
	if err != nil {
		panic(err)
	}
	input := bufio.NewScanner(envfile)
	for input.Scan() {
		line := strings.Split(input.Text(), "=")
		os.Setenv(line[0], line[1])
	}
}
func NewMyAudio(chatID int64, file RequestFileData, title string, duration int) AudioConfig {
	return AudioConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
		Title:    title,
		Duration: duration, // in seconds
	}
}

func NewMyVideo(chatID int64, file RequestFileData, title string, duration int) VideoConfig {
	return VideoConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
		Duration: duration,
		Caption:  title,
	}
}

func make_id() string {
	u2, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("failed to generate UUID: %v", err)
	}
	return u2.String()
}

func download_Youtube(url string, is_audio bool) (title string, duration int, fileid string) {
	// videoID := url
	// var stream io.ReadCloser
	client := youtube.Client{}
	var file *os.File
	video, err := client.GetVideo(url)
	if err != nil {
		panic(err)
	}
	duration = int(video.Duration.Seconds())
	title = video.Title
	formats := video.Formats
	fileid = make_id()
	if is_audio {
		formats = video.Formats.Type("audio/mp4")
		fileid += ".mp3"
		title += ".mp3"
		file, err = os.Create("media/" + fileid)
		if err != nil {
			panic(err)
		}

	} else {
		formats = video.Formats.WithAudioChannels() // only get videos with audio
		fileid += ".mp4"
		title += ".mp4"
		file, err = os.Create("media/" + fileid)
		if err != nil {
			panic(err)
		}

	}
	defer file.Close()
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(file, stream)
	if err != nil {
		panic(err)
	}
	return title, duration, fileid
}
func delete_old() {
	for {
		files, err := ioutil.ReadDir("media/")
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			if time.Since(file.ModTime()) > time.Minute*10 {
				os.Remove("media/" + file.Name())
			}
		}
		time.Sleep(5 * time.Minute)
	}
}
func main() {
	var (
		title    string
		fileid   string
		duration int
	)
	go delete_old()
	init_env()
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

			_, err := url.ParseRequestURI(update.Message.Text)
			if err == nil && update.Message.Text[:24] == "https://www.youtube.com/" {
				url_video = update.Message.Text
				messageid = update.Message.MessageID
				msg.ReplyMarkup = viauKeyboard
			}
			if _, err = bot.Send(msg); err != nil {
				panic(err)
			}
		} else if update.CallbackQuery != nil {
			chatcallbackId := update.CallbackQuery.Message.Chat.ID
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			if _, err := bot.Request(callback); err != nil {
				panic(err)
			}

			switch update.CallbackQuery.Data {
			case "Видео":
				msg_edit := tgbotapi.NewEditMessageTextAndMarkup(chatcallbackId, update.CallbackQuery.Message.MessageID, "Видео", videoKeyboard)
				is_audio = false
				if _, err := bot.Send(msg_edit); err != nil {
					panic(err)
				}

			case "Аудио":
				msg_edit := tgbotapi.NewEditMessageTextAndMarkup(chatcallbackId, update.CallbackQuery.Message.MessageID, "Аудио", audioKeyboard)
				is_audio = true
				if _, err := bot.Send(msg_edit); err != nil {
					panic(err)
				}

			// case "Изменить название файла":
			// 	if _, err := bot.Send(tgbotapi.NewEditMessageText(chatcallbackId, update.CallbackQuery.Message.MessageID, "Введите новое имя файла")); err != nil {
			// 		panic(err)
			// 	}
			// title = update.Message.Text
			// bot.Send(tgbotapi.NewEditMessageText(chatcallbackId, update.CallbackQuery.Message.MessageID, "Введите новое имя файла"))

			case "Продолжить в telegram":
				// exit from callback
				bot.Send(tgbotapi.NewChatAction(chatcallbackId, "typing"))
				msg_edit := tgbotapi.NewEditMessageText(chatcallbackId, update.CallbackQuery.Message.MessageID, "Загрузка")
				if _, err := bot.Send(msg_edit); err != nil {
					panic(err)
				}
				title, duration, fileid = download_Youtube(url_video, is_audio)
				if is_audio {
					bot.Send(tgbotapi.NewChatAction(chatcallbackId, "record_voice"))
					msg_final := NewMyAudio(chatcallbackId, tgbotapi.FilePath("media/"+fileid), title, duration)
					msg_final.ReplyToMessageID = messageid
					if _, err := bot.Send(msg_final); err != nil {
						panic(err)
					}
				} else {
					fmt.Println(url_video, is_audio)
					bot.Send(tgbotapi.NewChatAction(chatcallbackId, "upload_video"))
					msg_final := NewMyVideo(chatcallbackId, tgbotapi.FilePath("media/"+fileid), title, duration)
					msg_final.ReplyToMessageID = messageid
					if _, err := bot.Send(msg_final); err != nil {
						panic(err)
					}
				}
				bot.Send(tgbotapi.NewDeleteMessage(chatcallbackId, update.CallbackQuery.Message.MessageID))
				os.Remove("media/" + fileid)

			case "Назад":
				bot.Send(tgbotapi.NewDeleteMessage(chatcallbackId, update.CallbackQuery.Message.MessageID))
			}
		}
	}
}
