package model

import (
	"github.com/go-errors/errors"
	x "github.com/go-xorm/xorm"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"time"
	"wallet/redis"
)

//保存转出交易信息表
type trade struct {
	Id         int     `xorm:"not null INT(11) pk autoincr"`
	TradeId    int     `xorm:"not null default(0) int(11) comment('eth使用的交易id')"`
	Uid        string  `xorm:"not null default('') char(20) comment('uid')"`
	Addr       string  `xorm:"not null default('') varchar(255) comment('发送的地址')"`
	Num        float64 `xorm:"not null default(0.00000000) decimal(10,8) comment('交易的金额')"`
	CoinTypeId int     `xorm:"not null default(0) int(11) comment('转账的币种')"`
	Is_ok      int     `xorm:"not null default(0) tinyint(4) comment('是否已经交易完成 0表示未操作，1表示已经ok,2表示交易失败,3表示已经返还')"`
	Fee        float64 `xorm:"not null default(0.00000000) decimal(10,8) comment('转账需要的手续费')"`
	Time       int64   `xorm:"not null default(0) int(11) comment('交易的时间')"`
	tableName  string  `xorm:"-"`
	listTypeid int     `xorm:"-"` //上链的类型
}

type Trade struct {
	Id         int
	CoinTypeid int
}

//typeid为上链的链的类型（如比特，以太）
func NewTrade(typeid int) (t *trade) {
	t = new(trade)
	t.listTypeid = typeid
	char := ""
	if typeid == 1 {
		char = "btc"
	} else if typeid == 2 {
		char = "eth"
	} else {
		return nil
	}
	t.setTableName(char + "_trade")
	return
}

//创建表
func (t *trade) createTab() {
	err := xorm.MustDB().Table(t.tableName).Sync2(new(trade))
	if err != nil {
		faygo.Error(err.Error())
	}
}

func (t *trade) setTableName(tableName string) (tr *trade) {
	t.tableName = tableName
	return t
}

//查询出部分未转账成功的转账账单
func (t *trade) List(offset, limit int) (ids []int, err error) {
	engine := xorm.MustDB()
	rows, err := engine.Limit(limit, offset).Cols("id").Rows(t)
	if err != nil {
		err = SystemFail
		return
	}
	defer rows.Close()
	for rows.Next() {
		tr := new(trade)
		err = rows.Scan(tr)
		if err != nil {
			err = SystemFail
			return
		}
		ids = append(ids, tr.Id)
	}
	return
}

//增加一条交易记录
func (t *trade) Add(Typeid int, balance float64, fee float64, uid string, addr string, num float64) (err error) {
	//获取上链的类型
	coinType := AllCoinType.GetOneCoinInfoByTypeid(Typeid)
	if coinType == nil {
		err = errors.New("不存在的币种类型")
		return
	}

	engine := xorm.MustDB()
	//开启事务
	sess := engine.NewSession()
	err = sess.Begin()
	if err != nil {
		err = SystemFail
		return
	}
	//添加一条交易记录
	tr := new(trade)
	tr.Uid = uid
	tr.Addr = addr
	tr.Num = num
	tr.CoinTypeId = Typeid
	tr.Fee = fee
	tr.Time = time.Now().Unix()
	//如果是以太,需要全局一个id
	if coinType.ListType == 2 {
		//获取全局唯一的交易id
		id, err := DefaultAddrSerialNum.GetSerialNum()
		if err != nil {
			return err
		}
		tr.TradeId = id
	}
	n, err := sess.Table(t.tableName).Insert(tr)
	if err != nil || n == 0 {
		err = errors.New("添加交易记录失败")
		sess.Rollback()
		return
	}

	//如果是以太,需要全局维护一个id
	if coinType.ListType == 2 {
		//全局交易id累加
		res, err := sess.Exec("update addr_serial_num set serial_num=serial_num+1")
		if err != nil {
			err = SystemFail
			sess.Rollback()
			return err
		}
		n, err = res.RowsAffected()
		if err != nil || n == 0 {

			err = errors.New("添加交易记录失败")
			sess.Rollback()
			return err
		}
	}

	//扣除账户余额度
	coin := NewCoin(Typeid)
	if coin == nil {
		err = errors.New("不存在的币种类型")
		sess.Rollback()
		return
	}
	if balance-num < 0 {
		err = errors.New("账户余额不足")
		sess.Rollback()
		return
	}

	//修改账户（余额信息为刚刚查出的的余额度，因为正常转账情况下，除非修改成功，否则不存在余额出现变动）
	coin.Balance = balance - num
	n, err = sess.Table(coin.tableName).Where("uid=?", uid).And("balance=?", balance).Cols("balance").Update(coin)
	if err != nil {
		err = SystemFail
		sess.Rollback()
		return
	}
	if n == 0 {
		err = errors.New("添加交易记录失败")
		sess.Rollback()
		return
	}
	//添加一条转出记录日志
	coin_log := NewCoinLog(Typeid)
	if coin_log == nil {
		err = errors.New("不存在的币种类型")
		sess.Rollback()
		return
	}

	err = coin_log.AddLog(uid, 2, num, coin.Balance)
	if err != nil {
		sess.Rollback()
		return
	}

	sess.Commit()
	//放入到redis队列中的信息
	trade_data := Trade{}
	trade_data.Id = tr.Id
	trade_data.CoinTypeid = Typeid
	redis.New().Product("coin_transfer", trade_data)
	return
}

//根据id获取记录

//获取一条交易记录
func (t *trade) GetTradeById(id int) (tr *trade, err error) {
	engine := xorm.MustDB()
	tr = new(trade)
	b, err := engine.Table(t.tableName).Where("id=?", id).Get(tr)
	if err != nil {
		err = SystemFail
		return
	}
	if !b {
		tr = nil
	}
	return
}

//获取一条交易记录
func (t *trade) GetOneTrade() (tr *trade, err error) {
	engine := xorm.MustDB()
	tr = new(trade)
	b, err := engine.Where("is_ok", 0).Get(tr)
	if err != nil {
		err = SystemFail
		return
	}
	if !b {
		tr = nil
	}
	return
}

//进行上链
func (t *trade) PushList(coinTypeid int, addr string, num float64, id, tradeid int, fee float64) (err error) {
	//获取上链的类型(如果是不存在的类型直接不处理)
	coinType := AllCoinType.GetOneCoinInfoByTypeid(coinTypeid)
	if coinType == nil {
		return
	}
	//修改转账记录
	engine := xorm.MustDB()
	sess := engine.NewSession()
	err = sess.Begin()
	if err != nil {
		err = SystemFail
		return
	}
	trade_data := new(trade)
	trade_data.Is_ok = 1
	_, err = sess.Table(t.tableName).Where("trade_id=?", tradeid).And("id=?", id).Cols("is_ok").Update(trade_data)
	if err != nil {
		err = SystemFail
		sess.Rollback()
		return
	}

	//进行上链
	err = pushList(coinType.ListType, addr, num, tradeid, fee)
	//如果发送出错，将交易记录ok变成2，等待余额重置
	if err != nil {
		faygo.Debug("执行错误置2")
		trade_data.Is_ok = 2
		_, err = sess.Table(t.tableName).Where("trade_id=?", tradeid).And("id=?", id).Cols("is_ok").Update(trade_data)
		//如果修改成失败记录也失败，则回滚回初始状态，重新被查漏协程查出放到队列中执行，如果执行成功则ok置1，还是失败就会继续执行置2
		if err != nil {
			err = SystemFail
			sess.Rollback()
			return err
		}

	}
	sess.Commit()
	return
}

//eth补漏（因为eth发送交易成功，但是那边有可能会失败，导致这边记录是ok，那边失败未确认），直接发送账单重新确认
func (t *trade) PushListRepair(coinTypeid int, addr string, num float64, id int, fee float64) (err error) {
	//获取上链的类型(如果是不存在的类型直接不处理)
	coinType := AllCoinType.GetOneCoinInfoByTypeid(coinTypeid)
	if coinType == nil {
		return
	}
	//进行上链
	err = pushList(coinType.ListType, addr, num, id, fee)
	return
}

//获取未ok的记录
func (t *trade) GetNotOk(id, limit int) (list []Trade, err error) {
	engine := xorm.MustDB()
	var rows *x.Rows
	switch t.listTypeid {
	case 1:
		//比特类型
		rows, err = engine.Table(t.tableName).Cols("id", "coin_type_id").Where("is_ok=?", 0).Limit(limit).Rows(t)
	case 2:
		//eth
		rows, err = engine.Table(t.tableName).Cols("id", "coin_type_id").Where("is_ok=?", 0).And("id<?", id).Limit(limit).Rows(t)
	}

	if err != nil {
		faygo.Debug(err)
		err = SystemFail
		return
	}
	defer rows.Close()
	for rows.Next() {
		tra := new(trade)
		tr := Trade{}
		err = rows.Scan(tra)
		if err != nil {
			faygo.Debug(err)
			err = SystemFail
			return
		}
		tr.Id = tra.Id
		tr.CoinTypeid = tra.CoinTypeId
		list = append(list, tr)
	}
	return
}

//将需要需要返还余额的记录返还余额
func (t *trade) ReturnProfit(limit int) (err error) {
	engine := xorm.MustDB()
	rows, err := engine.Table(t.tableName).Where("is_ok=?", 2).Limit(limit).Rows(t)
	if err != nil {
		err = SystemFail
		return
	}
	defer rows.Close()
	var list []*trade
	for rows.Next() {
		tr := new(trade)
		err = rows.Scan(tr)
		if err != nil {
			continue
		}
		list = append(list, tr)
	}
	for _, v := range list {
		NewCoin(v.CoinTypeId).ReturnProfit(v.Id, v.TradeId, v.Uid, v.Num)
	}
	return
}
