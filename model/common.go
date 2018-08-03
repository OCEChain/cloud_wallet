package model

import (
	"encoding/json"
	"github.com/henrylee2cn/faygo"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type jsonData struct {
	Code    string      `json:"code"`
	Message interface{} `json:"message"`
}

func curl_get(url string, duration time.Duration) (data jsonData, err error) {
	client := &http.Client{Timeout: duration}
	res, err := client.Get(url)
	if err != nil {
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &data)
	return
}

func curl_post(u string, param_data string, duration time.Duration) (data jsonData, err error) {
	client := &http.Client{
		Timeout: duration,
	}

	resp, err := client.Post(u, "", strings.NewReader(param_data))
	if err != nil {
		return
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		faygo.Debug("读取body数据不错", err)
		return
	}
	err = json.Unmarshal(b, &data)
	return
}
