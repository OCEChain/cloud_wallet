package model

import (
	"errors"
	"fmt"
	"github.com/henrylee2cn/faygo"
	"strconv"
	"time"
	"wallet/config"
)

//以太坊转账
type eth_token_transfer struct {
}

var defaultEthTokenTransfer = new(eth_token_transfer)

//获取gasprice
func (e *eth_token_transfer) getGasPrice() (price string, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/eth/gasPrice"
	data, err := curl_get(url, time.Second*50)
	if err != nil {
		return
	}
	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("获取gasprice失败")
		return
	}
	//返回价格
	price = fmt.Sprintf("%v", data.Message)
	return
}

//交易签名
func (e *eth_token_transfer) sign(addr string, num float64, gasPrice string, nonce int, coin_addr string) (hex string, err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "not_online_url").String()
	url = url + "/eth-token/sign"

	param := fmt.Sprintf("%v|%v|%v|%v|%v", coin_addr, addr, gasPrice, nonce, strconv.FormatFloat(num, 'f', 8, 64))
	faygo.Debug(param)
	data, err := curl_post(url, param, time.Minute)

	if err != nil {
		return
	}
	//已经有了记录
	if data.Code == "666" {
		faygo.Debug(data.Message)
		err = HasEthId
		return
	}

	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("交易签名失败")
		return
	}
	hex = data.Message.(string)
	return
}

func (e *eth_token_transfer) send(sign_msg string) (err error) {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info(err)
		}
	}()
	url := config.GetConfig("list", "online_url").String()
	url = url + "/eth/send"
	data, err := curl_post(url, sign_msg, time.Minute)
	if err != nil {
		//faygo.Debug(data)
		return
	}
	faygo.Debug(data)
	//已经有了的记录
	if data.Code == "666" {
		faygo.Debug(666)
		return nil
	}
	if data.Code != "200" {
		faygo.Debug(data)
		err = errors.New("发送交易失败")
	}
	return
}
