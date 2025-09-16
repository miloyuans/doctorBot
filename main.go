package main

import (
	"doctorBot/menu"
	"doctorBot/tools"
	"fmt"
	"log"
	"strings"
	"time"

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

				// 验证原始 jobName 是否在 config.yaml 中
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

				// 验证环境是否合法
				var invalidEnvs []string
				for _, env := range environments {
					isValid := false
					for _, validEnv := range tools.ConfigData.ValidEnvironments {
						if env == validEnv {
							isValid = true
							break
						}
					}
					if !isValid {
						invalidEnvs = append(invalidEnvs, env)
					}
				}
				if len(invalidEnvs) > 0 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("环境 %s 不存在，是否输入错误或者联系运维配置环境", strings.Join(invalidEnvs, ", ")))
					bot.Send(msg)
					continue
				}

				var tag string
				var tagErr error
				// 如果命令是 gaming_manager_pre 且 projects=gaming-manager，执行一次镜像检测
				if jobName == "gaming_manager_pre" && params["projects"] == "gaming-manager" {
					ip := "13.251.90.38"
					port := "8000"
					imageName := "gaming-manager"
					branchName := fmt.Sprintf("%v", params["profile"])
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("正在从 %s:%s 获取 %s:%s 的镜像信息", ip, port, imageName, branchName))
					bot.Send(msg)
					tag, tagErr = tools.TriggerBuild(ip, port, imageName, branchName)
					if tagErr != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("获取镜像失败: %v，检查分支 %s 是否正确", tagErr, branchName))
						bot.Send(msg)
						return
					}
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("已获取到镜像信息 %s", tag))
					bot.Send(msg)
				}

				var resultMessages []string
				var errorMessages []string

				for i, env := range environments {
					// 复制参数并设置当前环境
					envParams := make(map[string]interface{})
					for k, v := range params {
						envParams[k] = v
					}
					envParams["environments"] = env

					// 使用原始 job 名称
					localJobName := jobName

					// 如果命令是 gaming_manager_pre 且 projects=gaming-manager，使用 gaming_manager_pre_push
					if jobName == "gaming_manager_pre" && envParams["projects"] == "gaming-manager" {
						localJobName = "gaming_manager_pre_push"
						if tag != "" {
							envParams["profile"] = tag
						}
					}

					// 触发 Jenkins Job
					jobURL, _ := tools.BuildJenkinsURL(tools.ConfigData.Jenkins.BaseURL, localJobName, envParams)
					log.Printf("环境 %s: 触发 Jenkins Job '%s'，URL: %s", env, localJobName, jobURL)
					statusCode, location := tools.TriggerJenkinsJob(localJobName, envParams, client)
					if statusCode != 201 {
						errorMessages = append(errorMessages, fmt.Sprintf("环境 %s: 触发 Jenkins Job '%s' 失败，状态码：%d，URL: %s", env, localJobName, statusCode, jobURL))
						continue
					}

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("环境 %s: 触发 Jenkins:'%s'，等待分配构建编号", env, localJobName))
					bot.Send(msg)

					buildNumber := tools.GetItemInfo(location, client)
					if buildNumber > 0 {
						resultMessages = append(resultMessages, fmt.Sprintf("环境 %s: 构建编号：%d   根据您所在的群,构建结果由不同的机器人通知", env, buildNumber))
					} else {
						errorMessages = append(errorMessages, fmt.Sprintf("环境 %s: 获取构建编号失败", env))
					}

					// 等待3秒再触发下一个环境
					if i < len(environments)-1 {
						time.Sleep(3 * time.Second)
					}
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
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "选择功能：")
				bot.Send(msg)
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