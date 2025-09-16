package tools

import (
	"fmt"
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gopkg.in/yaml.v2"
)

// Config 配置文件整体结构体
type Config struct {
	Jenkins struct {
		BaseURL  string `yaml:"base_url"`
		Username string `yaml:"username"`
		Token    string `yaml:"token"`
	} `yaml:"jenkins"`
	Base struct {
		Admin   string `yaml:"admin"`
		Private bool   `yaml:"private"`
	} `yaml:"base"`
	Telegram struct {
		Token           string  `yaml:"token"`
		AllowedGroupIDs []int64 `yaml:"allowed_group_ids"`
	} `yaml:"telegram"`
	ValidEnvironments []string `yaml:"valid_environments"`
	Jobs map[string]struct {
		Params []string `yaml:"params"`
		Help   string   `yaml:"help"`
	} `yaml:"jobs"`
}

var ConfigData Config

// 加载配置文件
func LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %v", err)
	}

	err = yaml.Unmarshal(data, &ConfigData)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	log.Println("配置文件加载成功")
	return nil
}

func GetClient(jenkinsUsername string, jenkinsToken string) *resty.Client {
	client := resty.New()
	client.SetBasicAuth(jenkinsUsername, jenkinsToken)
	// 启用调试模式
	client.SetDebug(false)
	return client
}

func GetBot(telegramToken string) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatalf("创建 Bot 失败: %v", err)
	}
	// 启用调试模式
	bot.Debug = false
	return bot
}

func InitMy() (client *resty.Client, bot *tgbotapi.BotAPI) {
	// 加载配置文件
	err := LoadConfig("conf/conf.yaml")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// Jenkins 配置
	client = GetClient(ConfigData.Jenkins.Username, ConfigData.Jenkins.Token)
	log.Printf("Jenkins Base URL: %s", ConfigData.Jenkins.BaseURL)

	// Telegram 配置
	bot = GetBot(ConfigData.Telegram.Token)
	log.Printf("Telegram Token: %s", ConfigData.Telegram.Token)

	return client, bot
}