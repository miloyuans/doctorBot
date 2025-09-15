// tools/buildUrl.go
package tools

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// 根据 Job 名称和参数组装 Jenkins API URL
func BuildJenkinsURL(jenkinsBaseURL string, job string, params map[string]interface{}) (string, map[string]string) {
	var urlStr string
	paramMap := make(map[string]string)

	if len(params) == 0 {
		// 无参数的 Job
		urlStr = fmt.Sprintf("%s/job/%s/build", jenkinsBaseURL, job)
	} else {
		// 有参数的 Job
		urlStr = fmt.Sprintf("%s/job/%s/buildWithParameters", jenkinsBaseURL, job)

		// 组装参数
		for key, value := range params {
			switch v := value.(type) {
			case string:
				// 对单个字符串参数值进行 URL 编码
				paramMap[key] = url.QueryEscape(v)
			case []string:
				// 将切片连接成字符串后再进行 URL 编码
				paramMap[key] = url.QueryEscape(strings.Join(v, ","))
			default:
				// 其他类型处理，转成字符串并编码
				paramMap[key] = url.QueryEscape(fmt.Sprintf("%v", v))
			}
		}
	}

	return urlStr, paramMap
}

// 使用正则表达式匹配并处理等号前后的空格
func CleanMessage(message string, botName string) string {
	// 处理 @botName
	if strings.Contains(message, "@") {
		messageParts := strings.SplitN(message, "@", 2) // 只分割一次，避免丢失后面参数
		if len(messageParts) > 1 {
			afterAt := strings.SplitN(messageParts[1], " ", 2) // 再次分割，找到 botName 后面的内容
			if afterAt[0] == botName {
				if len(afterAt) > 1 {
					message = messageParts[0] + " " + afterAt[1] // 拼接后面的参数
				} else {
					message = messageParts[0] // 仅去掉 @botName
				}
			}
		}
	}

	// 处理等号前后空格
	re := regexp.MustCompile(`\s*=\s*`)
	cleanedMessage := re.ReplaceAllString(message, "=")
	return cleanedMessage
}

// 解析用户输入的命令，返回 Job 名称和参数
func ParseCommand(message string, chatID int64) (string, map[string]interface{}) {
	// 去掉命令前缀 "/"
	command := strings.TrimPrefix(message, "/")
	parts := strings.SplitN(command, " ", 2)

	jobName := parts[0]
	params := make(map[string]interface{})

	if len(parts) > 1 {
		// 解析参数部分
		paramParts := strings.Split(parts[1], " ")
		for _, part := range paramParts {
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				key := kv[0]
				value := kv[1]

				// 特殊处理 environments 参数
				if key == "environments" && strings.Contains(value, ",") {
					params[key] = strings.Split(value, ",")
				} else if strings.Contains(value, ",") {
					params[key] = strings.Split(value, ",")
				} else {
					params[key] = value
				}
			}
		}
	}
	params["group"] = chatID
	return jobName, params
}

// 验证参数是否满足要求
func ValidateParams(required []string, provided map[string]interface{}) (bool, []string) {
	missing := []string{}
	for _, param := range required {
		if _, exists := provided[param]; !exists {
			missing = append(missing, param)
		}
	}
	return len(missing) == 0, missing
}