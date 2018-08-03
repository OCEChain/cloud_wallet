package model

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"strconv"
	"time"
	"wallet/config"
)

//比特币代币转账
type btc_token_transfer struct {
}

var defaultBtcTokenTransfer = new(btc_token_transfer)

//创建交易
func (b *btc_token_transfer) create(addr string, num float64, fee float64, propertyid int) (hex string, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()

	url := config.GetConfig("list", "online_url").String()
	url = url + "/btc-token/create"

	param := fmt.Sprintf("%v|%v|%v|%v", propertyid, addr, strconv.FormatFloat(num, 'f', 8, 64), strconv.FormatFloat(fee, 'f', 8, 64))

	data, err := curl_post(url, param, time.Second*30)
	if err != nil {
		return
	}
	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("创建交易失败")
		return
	}

	hex, ok := data.Message.(string)
	if !ok {
		err = errors.New("解析返回数据出错")
	}
	return
}

//交易签名
func (b *btc_token_transfer) sign(msg string) (hex string, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "not_online_url").String()
	url = url + "/btc-token/sign"
	data, err := curl_post(url, msg, time.Second*30)
	if err != nil {
		return
	}
	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("交易签名失败")
		return
	}
	//返回签名后的交易信息
	//获取其中的签名后的hex
	hex, ok := data.Message.(string)
	if !ok {
		err = errors.New("解析返回数据出错")
	}
	return
}

//发送交易
func (b *btc_token_transfer) send(sign_msg string) (err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/btc-token/send"
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

func (b *btc_token_transfer) getPoundage() (res map[string]interface{}, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/btc-token/fee"
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
