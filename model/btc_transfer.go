package model

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"strconv"
	"time"
	"wallet/config"
)

//比特币转账
type btc_transfer struct {
}

var defaultBtcTransfer = new(btc_transfer)

type createData struct {
	Complete bool   `json:"complete"`
	Final    bool   `json:"final"`
	Hex      string `json:"hex"`
}

var Return_push_list error = errors.New("区块链服务出错，放回队列")

//创建交易
func (b *btc_transfer) create(addr string, num float64, fee float64) (hex string, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/btc/create"

	param := fmt.Sprintf("%v|%v|%v", addr, strconv.FormatFloat(num, 'f', 8, 64), strconv.FormatFloat(fee, 'f', 8, 64))
	faygo.Debug(param)
	data, err := curl_post(url, param, time.Second*30)
	if err != nil {
		return
	}
	if data.Code != "200" {
		faygo.Debug(data.Message)
		err = errors.New("创建交易失败")
		return
	}

	//c_data := new(createData)
	//err = json.Unmarshal([]byte(fmt.Sprintf("%v", data.Message)), c_data)
	//if err != nil {
	//	faygo.Debug(err)
	//	faygo.Debug(data.Message)
	//	err = errors.New("解析返回数据出错")
	//	return
	//}
	//hex = c_data.Hex
	hex, ok := data.Message.(string)
	if !ok {
		err = errors.New("解析返回数据出错")
	}
	return
}

//交易签名
func (b *btc_transfer) sign(msg string) (hex string, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "not_online_url").String()
	url = url + "/btc/sign"
	data, err := curl_post(url, msg, time.Second*30)
	if err != nil {
		return
	}
	if data.Code != "200" {
		err = errors.New("交易签名失败")
		return
	}
	//返回签名后的交易信息
	//获取其中的签名后的hex
	//c_data := new(createData)
	//err = json.Unmarshal([]byte(fmt.Sprintf("%v", data.Message)), c_data)
	//if err != nil {
	//	faygo.Debug(data)
	//	err = errors.New("解析返回数据出错")
	//	return
	//}
	//hex = c_data.Hex
	hex, ok := data.Message.(string)
	if !ok {
		err = errors.New("解析返回数据出错")
	}
	return
}

//发送交易
func (b *btc_transfer) send(sign_msg string) (err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/btc/send"
	data, err := curl_post(url, sign_msg, time.Minute)
	if err != nil {
		return
	}

	if data.Code == "888" {
		err = Return_push_list
		return
	}
	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("发送交易失败")
	}
	return
}

func (b *btc_transfer) getPoundage() (res map[string]interface{}, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/btc/fee"
	data, err := curl_get(url, time.Second*30)
	if err != nil {
		return
	}
	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("请求失败")
		return
	}
	res = data.Message.(map[string]interface{})
	return
}
