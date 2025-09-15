package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 定义请求体结构体，BuildNumber 为指针类型
type RequestBody struct {
	ImageName  string  `json:"imageName"`
	BranchName string  `json:"branchName"`
}

// triggerBuild 方法，接收参数并发送 POST 请求
func TriggerBuild(ip string, port string, imageName string, branchName string)(string, error) {
	// 创建请求体数据
	data := RequestBody{
		ImageName:  imageName,
		BranchName: branchName,
	}

	// 将请求体数据编码成 JSON 格式
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "",fmt.Errorf("error encoding JSON: %w", err)
	}

	// 发送 POST 请求
	url := "http://" + ip + ":" + port + "/build"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "",fmt.Errorf("error creating request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "",fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应数据
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "",fmt.Errorf("error decoding response: %w", err)
	}

	// 判断返回的 code 是否为 0
	if code, ok := response["code"].(float64); ok && code == 0 {
		image := response["image"].(string)
		parts := strings.Split(image, ":")
		if len(parts) > 1 {
			tag := parts[1]
			return tag,nil
		} else {
			log.Printf("返回的镜像URL不对")
		}
	} 

	return "",fmt.Errorf("获取镜像失败")
}