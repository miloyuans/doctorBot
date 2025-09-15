package menu

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// 维护用户菜单栈
var userMenuStack = make(map[int64][]string)

// 发送主菜单
func SendMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	// 进入主菜单时，清空用户菜单栈
	userMenuStack[chatID] = []string{"main_menu"}

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛒 系统升级"),
			tgbotapi.NewKeyboardButton("💰 k8状态"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📞 新功能开发中"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "👋 选择功能:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// 发送系统升级菜单
func SendUpdateSystemMenu(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	// deletePreviousMessage(bot, chatID, messageID) // 删除旧消息

	// 记录用户进入此菜单
	userMenuStack[chatID] = append(userMenuStack[chatID], "update_menu")

	msg := tgbotapi.NewMessage(chatID, "你要升级哪个环境？")
	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("test", "test"),
		tgbotapi.NewInlineKeyboardButtonData("prod", "prod"),
		tgbotapi.NewInlineKeyboardButtonData("per", "per"),
		tgbotapi.NewInlineKeyboardButtonData("yfb", "yfb"),
		// tgbotapi.NewInlineKeyboardButtonData("返回", "go_back"),
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons[0], buttons[1]),
		tgbotapi.NewInlineKeyboardRow(buttons[2], buttons[3]),
		// tgbotapi.NewInlineKeyboardRow(buttons[4]),
	)
	bot.Send(msg)
}

// 发送 test 详情菜单
func SendTestMenu(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	deletePreviousMessage(bot, chatID, messageID) // 删除旧消息

	// 记录进入 test 菜单
	userMenuStack[chatID] = append(userMenuStack[chatID], "test_menu")

	msg := tgbotapi.NewMessage(chatID, "请选择具体操作：")
	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("操作 A", "action_a"),
		tgbotapi.NewInlineKeyboardButtonData("操作 B", "action_b"),
		tgbotapi.NewInlineKeyboardButtonData("返回", "go_back"),
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons[0], buttons[1]),
		tgbotapi.NewInlineKeyboardRow(buttons[2]),
	)
	bot.Send(msg)
}

// 处理回调事件
func HandleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	callbackQuery := update.CallbackQuery
	if callbackQuery != nil {
		chatID := callbackQuery.Message.Chat.ID
		messageID := callbackQuery.Message.MessageID

		switch callbackQuery.Data {
		case "test":
			SendTestMenu(bot, chatID, messageID)
		case "prod":
			deletePreviousMessage(bot, chatID, messageID)
			msg := tgbotapi.NewMessage(chatID, "你选择了 prod")
			bot.Send(msg)
		case "go_back":
			HandleGoBack(bot, chatID, messageID)
		default:
			msg := tgbotapi.NewMessage(chatID, "未知选项")
			bot.Send(msg)
		}

		// 回复 callback，防止按钮显示“正在处理”
		callback := tgbotapi.NewCallback(callbackQuery.ID, "")
		bot.Request(callback)
	}
}

// 处理返回逻辑
func HandleGoBack(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	stack, exists := userMenuStack[chatID]
	if !exists || len(stack) <= 1 {
		// 如果栈为空或已在主菜单
		deletePreviousMessage(bot, chatID, messageID)
		SendMainMenu(bot, chatID)
		return
	}

	// 弹出当前菜单，返回上一级
	userMenuStack[chatID] = stack[:len(stack)-1]
	previousMenu := userMenuStack[chatID][len(userMenuStack[chatID])-1]

	// 先删除上一个消息
	deletePreviousMessage(bot, chatID, messageID)

	// 返回上一级菜单
	switch previousMenu {
	case "main_menu":
		SendMainMenu(bot, chatID)
	case "update_menu":
		SendUpdateSystemMenu(bot, chatID, messageID)
	case "test_menu":
		SendTestMenu(bot, chatID, messageID)
	default:
		SendMainMenu(bot, chatID)
	}
}

// 删除上一条消息
func deletePreviousMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := bot.Request(deleteMsg); err != nil {
		log.Println("Error deleting message:", err)
	}
}
