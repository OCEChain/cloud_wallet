package handler

import (
	"encoding/json"
	"github.com/henrylee2cn/faygo"
	"os"
	"sync"
	"time"
	"wallet/model"
	"wallet/redis"
)

//管理当前账本将执行到id
type EthId struct {
	id int
	l  sync.RWMutex
}

func (e *EthId) SetId(id int) {
	e.l.Lock()
	defer e.l.Unlock()
	if e.id < id {
		e.id = id //下一次要交易的id
	}

}

func (e *EthId) GetId() (id int) {
	e.l.RLock()
	defer e.l.RUnlock()
	return e.id
}

var TradeEthId *EthId = new(EthId)

//手续费
type fee struct {
	ethFee string
	btcFee string
	l      sync.RWMutex
}

var coinFee *fee = new(fee)

func (f *fee) Set(typeid int, res string) {
	f.l.Lock()
	defer f.l.Unlock()
	switch typeid {
	case 1:
		f.btcFee = res
	case 2:
		f.ethFee = res
	}
}

func (f *fee) Get(typeid int) (res string) {
	f.l.RLock()
	defer f.l.RUnlock()
	switch typeid {
	case 1:
		res = f.btcFee
	case 2:
		res = f.ethFee
	}
	return
}

//开启协程
func init() {
	go func() {
		//程序一启动,从redis队列中读取那些去要转账的账单进行转账
		c := redis.Consumer("coin_transfer")
		trade_data := model.Trade{}
		for data := range c {
			err := json.Unmarshal([]byte(data), &trade_data)
			if err != nil {
				continue
			}

			//查询出该记录
			trade_model := model.NewTrade(trade_data.CoinTypeid)
			//说明这是不存在的类型，直接抛弃
			if trade_model == nil {
				continue
			}

			trade, err := trade_model.GetTradeById(trade_data.Id)
			if err != nil || trade.Is_ok == 1 || trade.Is_ok == 2 || trade.Is_ok == 3 {
				continue
			}

			err = trade_model.PushList(trade.CoinTypeId, trade.Addr, trade.Num, trade.Id, trade.TradeId, trade.Fee)
			//出错直接继续
			if err != nil {
				//faygo.Debug(err)
				//redis.New().Replay("coin_transfer", trade_data)
				continue
			}
			//faygo.Debug("当前执行到的id为", trade.TradeId)
			//设置当前执行到的
			TradeEthId.SetId(trade.Id + 1)

		}
	}()

	id, err := model.GetEthId()
	if err != nil {
		faygo.Info(err)
		//os.Exit(2)
	}
	faygo.Debug(id)
	//定时将执行失败的记录中的额度返还
	go func() {

		tick := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-tick.C:
				//查询出记录中的需要返回的
				model.NewTrade(1).ReturnProfit(10)
				model.NewTrade(2).ReturnProfit(10)
			}
		}
	}()

	go func() {
		//程序启动先获取两种币的手续费
		res, err := model.GetPoundage(1)
		if err != nil {
			faygo.Info(err)
			os.Exit(2)
		}
		coinFee.Set(1, res)
		res, err = model.GetPoundage(2)
		if err != nil {
			faygo.Info(err)
			os.Exit(2)
		}
		coinFee.Set(2, res)
		//定时去获取price
		tick := time.NewTicker(time.Second * 3)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				res, err = model.GetPoundage(1)
				if err == nil {
					coinFee.Set(1, res)
				}
				res, err = model.GetPoundage(2)
				if err == nil {
					coinFee.Set(2, res)
				}
			}
		}
	}()

	//开启一个协程，定时去将数据库中小于当前需要转账id的未确认记录拿出来重新执行一遍(避免执行失败后，将记录标记ok为2失败，重新执行）
	for i := 1; i < 3; i++ {
		go func(list_typeid int) {
			trade := model.NewTrade(list_typeid)
			tick := time.NewTicker(time.Second * 10)
			for {
				select {
				case <-tick.C:
					//faygo.Debug("开始查漏")
					var list []model.Trade
					var err error
					switch list_typeid {
					case 1:
						//如果是比特币类型的
						list, err = trade.GetNotOk(0, 20)
					case 2:
						//如果是eth类型的
						//获取当前将要执行的账本id
						id := TradeEthId.GetId()
						list, err = trade.GetNotOk(id, 20)
					}
					if err != nil {
						//faygo.Debug("查漏失败", err)
						continue
					}
					//faygo.Debug("查漏出来的结果为", list)
					l := len(list)

					for i := l - 1; i >= 0; i-- {
						//放入队列中
						//faygo.Debug("将未确认的放入队列中", list[i])
						redis.New().Replay("coin_transfer", list[i])
					}

				}
			}
		}(i)
	}

}
