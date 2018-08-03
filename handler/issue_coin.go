package handler

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"strconv"
	"sync"
	"time"
	"wallet/model"
)

type profit struct {
	uid    string
	num    float64
	typeid int //币种类型
}

//管理需要发币的账号
type queue struct {
	l sync.RWMutex
	c map[string]chan bool
}

var Queue *queue
var Profit_queue chan *profit

func init() {
	Queue = new(queue)
	Queue.c = make(map[string]chan bool)
	Profit_queue = make(chan *profit, 1000)
	//开启协程去收益队列中将每个发币的记录给记录到数据库中
	go func() {
		for data := range Profit_queue {
			coin := model.NewCoin(data.typeid)
			err := coin.AddProfit(data.uid, data.num)
			if err != nil {
				continue
			}
		}
	}()
}

func (q *queue) Is_exist(uid string) (ok bool) {
	q.l.RLock()
	defer q.l.RUnlock()
	_, ok = q.c[uid]
	return
}

//发币
func (q *queue) Issue(uid string) {
	//如果已经存在该发币的协程,直接返回
	if q.Is_exist(uid) {
		return
	}
	q.l.Lock()
	defer q.l.Unlock()
	q.c[uid] = make(chan bool)
	faygo.Debug("开始发币")
	//获取所有的币种的信息
	coinTypeList := model.AllCoinType.GetAllInfo()
	for _, v := range coinTypeList {
		//给每个币种开协程进行发币
		//跑出一个协程进行发币
		go func(uid string, ch chan bool, coinType *model.CoinType) {
			//假设是5分钟发一次币
			tick := time.NewTicker(time.Second * time.Duration(coinType.Issue_time))
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					//计算当前收益
					faygo.Debug("当前的币种id为", coinType.Id)
					coin, err := getProfit(uid, coinType.Id)
					if err != nil {
						continue
					}
					//faygo.Debug("当前发的收益为：", coin)
					//到了发币的时间,将要发的币的放到队列中（串型化，避免数据库阻塞）
					p := new(profit)
					p.typeid = coinType.Id
					p.num = coin
					p.uid = uid
					Profit_queue <- p
				case <-ch:
					//关闭信号的时候，关闭发币协程
					return
				}
			}
		}(uid, q.c[uid], v)
	}

}

//停止某个用户发币
func (q *queue) Stop(uid string) {
	q.l.Lock()
	defer q.l.Unlock()
	ch, ok := q.c[uid]
	if !ok {
		return
	}
	//关闭管道，让发币协程退出
	close(ch)
	//删除发币队列中的账号
	delete(q.c, uid)
}

//当添加虚拟币或者是禁用虚拟币的时候，重置所有的发币程序
func (q *queue) ReIssue() bool {
	for uid, _ := range q.c {
		//关闭当前的发币程序
		q.Stop(uid)
		//重新载入发币程序
		q.Issue(uid)
	}
	return true
}

//计算收益
func getProfit(uid string, coin_typeid int) (profit float64, err error) {
	//获取算力
	cal := ALlCal.GetCal(uid)
	total := ALlCal.GetTotal()
	cal_bi := cal / total
	//获取币种信息
	coin_type := model.AllCoinType.GetOneCoinInfoByTypeid(coin_typeid)
	if coin_type == nil {
		err = errors.New("不存在的币种类型")
		return
	}
	//获取时间间隔
	num := float64(coin_type.Coin_time / coin_type.Issue_time)
	if coin_type.Coin_price == 0 || num == 0 {
		profit = 0
		return
	}

	profit = cal_bi * coin_type.Coin_price / num
	if profit > coin_type.Colin_limit {
		profit = coin_type.Colin_limit
	}
	return
}

// 浮点型转字符串
func FloatToString(f float64) string {
	return fmt.Sprintf("%0.2f", f)
}

// 字符串转浮点型
func StringToFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}

// 浮点数舍入
func FloatToFloat(f float64) float64 {
	f, _ = StringToFloat(FloatToString(f))
	return f
}

//避免出现0.999999999这种数据
func FloatAddFloat(a, b float64) float64 {
	return FloatToFloat((a*100 + b*100) / 100)
}

//避免出现0.999999999这种数据
func FloatCutFloat(a, b float64) float64 {
	return FloatToFloat((a*100 - b*100) / 100)
}
