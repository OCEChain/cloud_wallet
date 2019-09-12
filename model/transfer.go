package model

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wallet/config"
)

/*
* 进行上链
* list_type 币种类型 1比特币 2以太币 3比特代币 4以太代币
* addr 转账地址
* num 转账数量
* id nouceID 用于以太币或者以太代币
* propertyid 用户比特代币转账
* coin_addr 合约地址 用于以太代币转账
 */
func pushList(list_type int, addr string, num float64, id int, fee float64, propertyid int, coin_addr string) error {
	switch list_type {
	case 1:
		faygo.Debug("转比特币")
		hex, err := defaultBtcTransfer.create(addr, num, fee)
		if err != nil {
			return err
		}
		sign_hex, err := defaultBtcTransfer.sign(hex)
		if err != nil {
			return err
		}

		return defaultBtcTransfer.send(sign_hex)
	case 2:
		faygo.Debug("转以太币")
		price, err := defaultEthTransfer.getGasPrice()
		if err != nil {
			return err
		}
		sign_hex, err := defaultEthTransfer.sign(addr, num, price, id)
		if err == HasEthId {
			return nil
		} else if err != nil {
			return err
		}
		return defaultEthTransfer.send(sign_hex)
	case 3:
		faygo.Debug("转比特代币")
		//比特代币
		hex, err := defaultBtcTokenTransfer.create(addr, num, fee, propertyid)
		sign_hex, err := defaultBtcTokenTransfer.sign(hex)
		if err != nil {
			return err
		}

		return defaultBtcTokenTransfer.send(sign_hex)
	case 4:
		faygo.Debug("转以太代币")
		faygo.Debug(addr, num, id, coin_addr)
		//以太代币
		price, err := defaultEthTokenTransfer.getGasPrice()
		if err != nil {
			return err
		}
		sign_hex, err := defaultEthTokenTransfer.sign(addr, num, price, id, coin_addr)
		if err == HasEthId {
			return nil
		} else if err != nil {
			return err
		}

		return defaultEthTokenTransfer.send(sign_hex)
	default:
		return nil
	}
}

func GetEthId() (id int, err error) {
	url := config.GetConfig("list", "online_url").String()
	url = url + "/eth/count"
	data, err := curl_get(url, time.Second*50)
	if err != nil {
		return
	}
	if data.Code != "200" {
		err = errors.New("获取不到id")
		return
	}
	id_str := data.Message.(string)
	id, _ = strconv.Atoi(id_str)
	return
}

func GetPoundage(ListType int) (res string, err error) {
	switch ListType {
	case 1, 3:
		//比特
		data, err1 := defaultBtcTransfer.getPoundage()
		if err != nil {
			err = err1
			return
		}
		b, err2 := json.Marshal(data)
		if err != nil {
			err = err2
			return
		}
		res = string(b)
		//faygo.Debug(res)
	case 2, 4:
		res, err = defaultEthTransfer.getGasPrice()

	}
	return
}

//获取币种的价格
func GetPrice(coin string) (data string, err error) {
	char := strings.ToUpper(coin)
	url := config.GetConfig("list", "price_url").String()
	url = url + "/price/" + char
	client := &http.Client{Timeout: time.Second * 10}
	res, err := client.Get(url)
	if err != nil {
		err = errors.New("请求失败")
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	data = string(b)
	return
}
