// tools/getItemInfo.go
package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-resty/resty/v2"
)

func GetItemInfo(location string, client *resty.Client) int {
	apiURL := fmt.Sprintf("%s/api/json", location)

	for {
		resp, err := client.R().Get(apiURL)
		if err != nil {
			log.Printf("查询队列状态失败 (%s): %v", location, err)
			return 0
		}
		var queueStatus map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &queueStatus); err != nil {
			log.Printf("解析响应失败 (%s): %v", location, err)
			return 0
		}

		if executable, ok := queueStatus["executable"].(map[string]interface{}); ok {
			buildNumber := int(executable["number"].(float64))
			log.Printf("构建已分配编号 (%s): %d", location, buildNumber)
			return buildNumber
		}

		log.Printf("任务仍在队列中 (%s)，等待分配构建编号...", location)
		time.Sleep(5 * time.Second)
	}
}
func TriggerJenkinsJob(job string, params map[string]interface{}, client *resty.Client) (int, string) {

	// 获取 URL 和参数
	jobURL, paramMap := BuildJenkinsURL(ConfigData.Jenkins.BaseURL, job, params)
	log.Printf("jobURL: %s paramMap: %v", jobURL, paramMap)

	// 发送请求
	req := client.R()
	if len(paramMap) > 0 {
		req.SetFormData(paramMap)
	}

	resp, err := req.Post(jobURL)
	if err != nil {
		log.Printf("触发 Jenkins Job 失败: %v", err)
		return resp.StatusCode(), ""
	}

	location := resp.Header().Get("Location")
	return resp.StatusCode(), location
}