package main

import (
	// "fmt"
	"bufio"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func main() {
	// t.me/mytogo_bot
	init_env()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		// msg.ReplyToMessageID = update.Message.MessageID
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	}

}
