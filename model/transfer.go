package model

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"user/config"
)

//进行上链
func pushList(list_type int, addr string, num float64, id int, fee float64) error {
	switch list_type {
	case 1:
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
	case 1:
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
	case 2:
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
