// main.go
package main

import (
	"doctorBot/menu"
	"doctorBot/tools"
	"fmt"
	"log"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	client, bot := tools.InitMy()
	botName := bot.Self.UserName
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Printf("机器人已启动，用户名: %s", botName)

	for update := range updates {
		if update.CallbackQuery != nil {
			menu.HandleCallbackQuery(bot, update)
		}

		if update.Message == nil {
			continue
		}

		if update.Message.NewChatMembers != nil || update.Message.LeftChatMember != nil {
			continue
		}

		if tools.ConfigData.Base.Private {
			if !tools.IsGroupMessage(update.Message.Chat.Type) {
				tools.SendMessage(bot, update.Message.Chat.ID, "私聊无效，请在指定群组发送命令，或联系 "+tools.ConfigData.Base.Admin+" 授权")
				continue
			}
			if !tools.IsAllowedGroup(update.Message.Chat.ID, tools.ConfigData.Telegram.AllowedGroupIDs) {
				tools.SendMessage(bot, update.Message.Chat.ID, "这个群组为非授权群组，请联系"+tools.ConfigData.Base.Admin+"授权")
				continue
			}
		}

		message := strings.TrimSpace(update.Message.Text)
		message = tools.CleanMessage(message, botName)
		log.Printf("用户输入[%s] %s", update.Message.From.UserName, message)

		if strings.HasPrefix(message, "/") {
			excludedCommands := map[string]bool{
				"/help": true,
				"/menu": true,
			}
			if !excludedCommands[message] {
				jobName, params := tools.ParseCommand(message, update.Message.Chat.ID)
				log.Println("参数:", params)

				jobConfig, ok := tools.ConfigData.Jobs[jobName]
				if !ok {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("不支持的命令 '%s'，请检查命令或使用 /help 查看支持的 Job。", jobName))
					bot.Send(msg)
					continue
				}

				valid, missing := tools.ValidateParams(jobConfig.Params, params)
				if !valid {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("参数不完整，缺少: %v", missing))
					bot.Send(msg)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, tools.ConfigData.Jobs[jobName].Help)
					bot.Send(msg)
					continue
				}

				// 处理多环境
				environments, isMultiEnv := params["environments"].([]string)
				if !isMultiEnv {
					environments = []string{fmt.Sprintf("%v", params["environments"])}
				}

				var wg sync.WaitGroup
				results := make(chan string, len(environments))
				errors := make(chan string, len(environments))

				for _, env := range environments {
					wg.Add(1)
					go func(env string) {
						defer wg.Done()

						// 动态调整 job 名称和参数
						envParams := make(map[string]interface{})
						for k, v := range params {
							envParams[k] = v
						}
						envParams["environments"] = env

						/*/ 根据环境选择 job 名称
						localJobName := jobName
						if env == "eks-yfb" && !strings.HasSuffix(jobName, "_pre") {
							localJobName = jobName + "_pre"
						} else if env != "eks-yfb" && strings.HasSuffix(jobName, "_pre") {
							localJobName = strings.TrimSuffix(jobName, "_pre")
						}*/

						// 处理特殊 job（如 games_cocos 和 gaming_manager_pre）
						if localJobName == "games_cocos" {
							ip := "52.74.65.246"
							port := "8000"
							imageName := "gaming-cocos"
							branchName := fmt.Sprintf("%v", envParams["TAG"])
							tag, err := tools.TriggerBuild(ip, port, imageName, branchName)
							if err != nil {
								errors <- fmt.Sprintf("环境 %s: 获取镜像失败: %v", env, err)
								return
							}
							results <- fmt.Sprintf("环境 %s: 已获取到镜像信息 %s", env, tag)
							localJobName = "games_cocos_push"
							envParams["TAG"] = tag
						} else if localJobName == "gaming_manager_pre" {
							ip := "13.251.90.38"
							port := "8000"
							imageName := "gaming-manager"
							branchName := fmt.Sprintf("%v", envParams["profile"])
							tag, err := tools.TriggerBuild(ip, port, imageName, branchName)
							if err != nil {
								errors <- fmt.Sprintf("环境 %s: 获取镜像失败: %v", env, err)
								return
							}
							results <- fmt.Sprintf("环境 %s: 已获取到镜像信息 %s", env, tag)
							localJobName = "gaming_manager_pre_push"
							envParams["profile"] = tag
						}

						// 触发 Jenkins Job
						statusCode, location := tools.TriggerJenkinsJob(localJobName, envParams, client)
						if statusCode != 201 {
							errors <- fmt.Sprintf("环境 %s: 触发 Jenkins Job '%s' 失败，状态码：%d", env, localJobName, statusCode)
							return
						}

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("环境 %s: 触发 Jenkins:'%s'，等待分配构建编号", env, localJobName))
						bot.Send(msg)

						buildNumber := tools.GetItemInfo(location, client)
						if buildNumber > 0 {
							results <- fmt.Sprintf("环境 %s: 构建编号：%d", env, buildNumber)
						} else {
							errors <- fmt.Sprintf("环境 %s: 获取构建编号失败", env)
						}
					}(env)
				}

				// 等待所有任务完成
				go func() {
					wg.Wait()
					close(results)
					close(errors)
				}()

				// 收集结果
				var resultMessages []string
				var errorMessages []string
				for result := range results {
					resultMessages = append(resultMessages, result)
				}
				for errMsg := range errors {
					errorMessages = append(errorMessages, errMsg)
				}

				// 发送结果
				if len(resultMessages) > 0 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, strings.Join(resultMessages, "\n"))
					bot.Send(msg)
				}
				if len(errorMessages) > 0 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "错误：\n"+strings.Join(errorMessages, "\n"))
					bot.Send(msg)
				}

			} else if message == "/help" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "请从对话窗口直接输入 / 查看命令")
				bot.Send(msg)
			} else if message == "/menu" {
				menu.SendMainMenu(bot, update.Message.Chat.ID)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "没有这个命令")
				bot.Send(msg)
			}
		} else if message == "🛒 系统升级" {
			menu.SendUpdateSystemMenu(bot, update.Message.Chat.ID, update.Message.MessageID)
		}
	}
}