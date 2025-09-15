// permissions.go
package tools

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// 判断消息是否来自群组
func IsGroupMessage(chatType string) bool {
	return chatType == "supergroup" || chatType == "group"
}

// 检查消息是否来自允许的群组
func IsAllowedGroup(chatID int64, groupIDs []int64) bool {
	for _, id := range groupIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

// 返回错误消息
func SendMessage(bot *tgbotapi.BotAPI, chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	bot.Send(msg)
}
