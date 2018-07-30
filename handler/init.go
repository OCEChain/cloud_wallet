package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"wallet/model"
)

func test() {
	//获取所有币种的id
	url := "http://api.coinmarketcap.com/v2/listings/"
	client := &http.Client{Timeout: time.Second * 5}
	res, err := client.Get(url)
	if err != nil {
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		fmt.Println(err)
		return
	}
	list := data["data"].([]interface{})
	add(list)

}

func add(list []interface{}) {
	err_list := []interface{}{}
	for _, v := range list {
		//插入到数据库中
		v_data := v.(map[string]interface{})
		fmt.Println(v_data)
		id := int(v_data["id"].(float64))
		char := v_data["symbol"].(string)
		err := model.DefaultCoinIds.Add(id, char)
		if err != nil {
			err_list = append(err_list, v)
		}
	}
	if len(err_list) > 0 {
		add(err_list)
	}
}
