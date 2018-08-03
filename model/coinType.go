package model

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"user/config"
	"wallet/redis"
)

//管理各种币种的信息
type coin_type struct {
	l            sync.RWMutex
	coinInfo     map[string]*CoinType //key为币种符号名称（如 btc）
	coinInfoById map[int]*CoinType    //以币种类型id为key
	coinList     []*CoinType
	auto_price   chan int //控制是否退出自动获取价格的协助程序
}

var AllCoinType *coin_type

//初始化各种币种模型
func init() {
	err := xorm.MustDB().Table(CoinTypeTABLE).Sync2(new(CoinType))
	if err != nil {
		faygo.Error(err.Error())
	}
	AllCoinType = new(coin_type)
	err = AllCoinType.Init()
	if err != nil {
		faygo.Info(err)
		os.Exit(2)
	}
	//初始化根据币种类型自动构造各种币的表
	coinTypeList := AllCoinType.GetAllInfo()
	for _, v := range coinTypeList {
		NewCoin(v.Id).createTab()
		NewCoinLog(v.Id).createTab()

		//如果开通转账功能
		if v.ListType != 0 {
			NewTrade(v.Id).createTab()
		}
	}
}

//获取某种币种的信息
func (c *coin_type) GetOneCoinInfoByChar(coinChar string) (coinType *CoinType) {
	c.l.RLock()
	defer c.l.RUnlock()
	coinType, ok := c.coinInfo[coinChar]
	if !ok {
		return nil
	}
	return
}

//添加某种币种的信息
func (c *coin_type) Add(coinType *CoinType) {
	c.l.Lock()
	defer c.l.Unlock()
	c.coinInfo[coinType.Coin_char] = coinType
	c.coinInfoById[coinType.Id] = coinType
}

//获取所有的币种信息
func (c *coin_type) GetAllInfo() map[string]*CoinType {
	return c.coinInfo
}

func (c *coin_type) GetAllInfoList() []*CoinType {
	return c.coinList
}

//初始化各种币种信息
func (c *coin_type) Init() (err error) {
	//重新获取所有币种的各项信息
	coinTypeModel := new(CoinType)
	list, listById, typelist, _, err := coinTypeModel.GetCoinType()
	if err != nil {
		return
	}

	auto_chan := make(chan int) //重新make一个chan取管理那些即将新开去自动获取价格的协程
	//重新初始化各项信息
	c.coinInfo = list
	c.coinInfoById = listById
	c.coinList = typelist
	//如果币种为自动获取币种价格的话，则启动协程定时(5分钟一次)去刷新获取
	for k, v := range typelist {
		if v.Get_price == 1 {
			//如果是自动获取价格
			price, err := c.AutoGetPrice(v.Coin_char)
			if err != nil {
				return err
			}

			typelist[k].Coin_price, _ = Tofix(price, 2)
			//定时获取价格
			go func(coin_type *CoinType, close_chan chan int) {
				tick := time.NewTicker(time.Minute * 5)
				for {
					select {
					case <-tick.C:
						//如果是自动获取价格
						price, err := c.AutoGetPrice(coin_type.Coin_char)
						if err != nil {
							continue
						}
						coin_type.Coin_price = price
					case <-close_chan:
						return
					}

				}
			}(typelist[k], auto_chan)
		}
	}
	if c.auto_price != nil {
		close(c.auto_price) //关闭之前自动获取价格的协程
	}
	faygo.Debug(typelist[0])
	c.auto_price = auto_chan
	return
}

//根据币种类型id获取币种信息
func (c *coin_type) GetOneCoinInfoByTypeid(typeid int) (coinType *CoinType) {
	c.l.RLock()
	defer c.l.RUnlock()
	coinType, ok := c.coinInfoById[typeid]
	if !ok {
		return nil
	}
	return
}

//自动获取价格
func (c *coin_type) AutoGetPrice(coin_char string) (price float64, err error) {
	price_str, err := redis.New().Get(coin_char + "_price")
	if err != nil {
		return
	}
	if price_str != "" {
		price, _ = strconv.ParseFloat(price_str, 64)
		return
	}

	//通过币种字符获取可供查询的币id
	id, err := DefaultCoinIds.GetIdByChar(coin_char)
	if err != nil {
		return
	}
	url := config.GetConfig("price", "url").String() + "/v2/ticker/" + fmt.Sprintf("%v", id) + "?convert=CNY"
	//通过id获取价格
	client := &http.Client{Timeout: time.Second * 10}
	res, err := client.Get(url)
	if err != nil {
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		err = errors.New("自动获取价格,解析数据出错,错误信息为" + err.Error())
		return
	}
	price_data, ok := data["data"]
	if !ok {
		err = errors.New("自动获取价格出错，获取不到data")
		return
	}
	price_d := price_data.(map[string]interface{})
	quotes, ok := price_d["quotes"]
	if !ok {
		err = errors.New("自动获取价格出错，获取不到quotes")
		return
	}
	quotes_data := quotes.(map[string]interface{})
	cny, ok := quotes_data["CNY"]
	if !ok {
		err = errors.New("自动获取价格出错，获取不到CNY")
		return
	}
	cny_data := cny.(map[string]interface{})
	price = cny_data["price"].(float64)
	redis.New().Set(coin_char+"_price", price, "600")
	return
}

//保留n位小数
func Tofix(f float64, n int) (res float64, err error) {
	format := "%." + strconv.Itoa(n) + "f"
	float_str := fmt.Sprintf(format, f)
	res, err = strconv.ParseFloat(float_str, 64)
	return
}
