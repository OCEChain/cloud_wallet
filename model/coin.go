package model

import (
	"encoding/json"
	"errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"wallet/config"
)

type coin struct {
	Uid            string  `xorm:"not null unique default('') char(20) comment('用户uid')"`
	Profit         float64 `xorm:"not null default(0.00000000) decimal(10,8) comment('累计收益')"`
	Receive_profit float64 `xorm:"not null default(0.00000000) decimal(10,8) comment('可领取的收益')"`
	Balance        float64 `xorm:"not null default(0.00000000) decimal(10,8) comment('余额')"`
	tableName      string  `xorm:"-"`
	tableLogName   string  `xorm:"-"`
	coinType       int     `xorm:"-"`
}

//构造函数
func NewCoin(typeid int) (c *coin) {
	c = new(coin)
	c.coinType = typeid
	coinType := AllCoinType.GetOneCoinInfoByTypeid(typeid)
	if coinType == nil {
		return nil
	}
	c.setTableName(coinType.Coin_char)

	return
}

//创建表
func (c *coin) createTab() {
	err := xorm.MustDB().Table(c.tableName).Sync2(new(coin))
	if err != nil {
		faygo.Error(err.Error())
	}
}

func (c *coin) setTableName(tableName string) (co *coin) {
	c.tableName = tableName
	c.tableLogName = tableName + "_log"
	return c
}

//增加收益
func (c *coin) AddProfit(uid string, profit float64) (err error) {
	err = c.InitUidCoin(uid)
	if err != nil {
		return
	}
	engine := xorm.MustDB()
	_, err = engine.Exec("update "+c.tableName+" set profit=profit+?,receive_profit=receive_profit+? where uid=?", profit, profit, uid)
	if err != nil {
		err = SystemFail
	}
	return
}

//领取收益(将可领取的收益清空，并且记录到一张表中)
func (c *coin) Receive(uid string) (err error) {
	engine := xorm.MustDB()
	sess := engine.NewSession()
	defer sess.Close()
	err = sess.Begin()
	if err != nil {
		err = SystemFail
		return
	}
	//查询出可领取的数目
	var receive float64
	co := new(coin)
	has, err := sess.Table(c.tableName).Where("uid=?", uid).Cols("receive_profit", "balance").Get(co)
	if err != nil {
		err = SystemFail
		sess.Rollback()
		return
	}

	if !has {
		err = errors.New("不存在的记录")
		sess.Rollback()
		return
	}
	receive = co.Receive_profit
	balance := co.Balance
	if receive == 0 {
		sess.Rollback()
		return
	}

	//扣除数目,并且增加余额
	res, err := sess.Exec("update "+c.tableName+" set receive_profit=receive_profit-?,balance=balance+? where uid=?", receive, receive, uid)
	if err != nil {
		err = SystemFail
		sess.Rollback()
		return
	}
	_, err = res.RowsAffected()
	if err != nil {
		err = SystemFail
		sess.Rollback()
		return
	}

	//增加一条领取记录
	coinlog := NewCoinLog(c.coinType)
	err = coinlog.setTableName(c.tableLogName).AddLog(sess, uid, 1, receive, balance+receive)
	if err != nil {
		sess.Rollback()
		err = SystemFail
		return
	}
	sess.Commit()
	return
}

//获取用户btc账户信息
func (c *coin) GetInfo(uid string) (err error) {
	engine := xorm.MustDB()
	has, err := engine.Table(c.tableName).Where("uid=?", uid).Get(c)
	if err != nil {
		err = SystemFail
		return
	}
	//如果不存在则创建一条
	if !has {
		c.Uid = uid
		err = c.create(uid)
		if err != nil {
			err = SystemFail
		}
	}
	return
}

//初始化uid的币种记录
func (c *coin) InitUidCoin(uid string) (err error) {
	engine := xorm.MustDB()
	has, err := engine.Table(c.tableName).Where("uid=?", uid).Exist()
	if err != nil {
		err = SystemFail
	}
	if has {
		return
	}
	err = c.create(uid)
	return
}

//创建一条记录
func (c *coin) create(uid string) (err error) {
	engine := xorm.MustDB()
	c.Uid = uid
	n, err := engine.Table(c.tableName).Insert(c)
	if err != nil {
		return
	}
	if n == 0 {
		err = errors.New("创建失败")
	}
	return
}

//币种返还余额度
func (c *coin) ReturnProfit(id, tradeid int, uid string, num float64) {
	coinType := AllCoinType.GetOneCoinInfoByTypeid(c.coinType)
	if coinType == nil {
		return
	}
	engine := xorm.MustDB()
	sess := engine.NewSession()
	err := sess.Begin()
	_, err = sess.Exec("update "+c.tableName+" set balance=balance+? where uid=?", num, uid)
	if err != nil {

		sess.Rollback()
		return
	}

	//将记录设置为已经返回余额度
	trade := NewTrade(coinType.Id)
	trade_table_name := trade.tableName
	trade.Is_ok = 3
	n, err := sess.Table(trade_table_name).Where("id=?", id).Update(trade)
	if err != nil || n == 0 {
		faygo.Debug(err)
		sess.Rollback()
		return
	}
	//如果是以太
	if coinType.ListType == 2 || coinType.ListType == 4 {

		//将tradeid返回，以备下次使用
		ids := new(Ids)
		ids.Is_use = 0
		n, err = sess.Where("id=?", tradeid).Cols("is_use").Update(ids)
		if err != nil || n == 0 {
			faygo.Debug(tradeid)
			faygo.Debug(err)
			sess.Rollback()
			return
		}
	}

	var balance float64
	//查询出最新的余额
	has, err := sess.Table(c.tableName).Where("uid=?", uid).Cols("balance").Get(&balance)
	if err != nil || !has {
		faygo.Debug(err)
		sess.Rollback()
		return
	}
	//增加一条返还失败返还记录
	coinLog := NewCoinLog(coinType.Id)
	coinLog.Num = num
	coinLog.Typeid = 3
	coinLog.Uid = uid
	coinLog.Balance = balance
	coinLog.Time = time.Now().Unix()
	n, err = sess.Table(coinLog.tableName).Insert(coinLog)
	if err != nil || n == 0 {
		sess.Rollback()
		return
	}

	sess.Commit()

}

//获取当前币种的账户总数
func (c *coin) Count() (count int64, err error) {
	engine := xorm.MustDB()
	count, err = engine.Table(c.tableName).Count()
	if err != nil {
		err = SystemFail
	}
	return
}

type coinAccount struct {
	*coin
	Account interface{}
}

//获取当前币种的账户列表
func (c *coin) List(uid string, offset, limit int) (account_list []*coinAccount, err error) {
	engine := xorm.MustDB()
	sess := engine.Table(c.tableName).Limit(limit, offset)
	if uid != "" {
		sess = sess.Where("uid=?", uid)
	}
	rows, err := sess.Rows(c)
	if err != nil {
		faygo.Debug(err)
		err = SystemFail
		return
	}
	defer rows.Close()
	uids := []string{}
	var list []*coin
	for rows.Next() {
		co := new(coin)
		err = rows.Scan(co)
		if err != nil {
			err = SystemFail
			return
		}
		list = append(list, co)
		uids = append(uids, co.Uid)
	}
	url := config.GetConfig("user", "url").String() + "/admin/getaccount"
	param := make(map[string]string)
	param["uids"] = strings.Join(uids, ",")
	faygo.Debug(param)
	data, err := post(url, param, time.Second*5)
	if err != nil {
		faygo.Debug(err)
		err = SystemFail
		return
	}
	if data.Code != 200 {
		faygo.Debug(data)
		err = errors.New("获取数据失败")
		return
	}
	post_data := data.Data.(map[string]interface{})
	account_list = make([]*coinAccount, len(list))
	for k, v := range list {
		a := new(coinAccount)
		a.coin = v

		account, ok := post_data[v.Uid]
		if ok {
			a.Account = account
		}
		account_list[k] = a
	}

	return
}

type json_Data struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

func post(u string, param map[string]string, duration time.Duration) (data json_Data, err error) {
	client := &http.Client{
		Timeout: duration,
	}
	p := url.Values{}
	for k, v := range param {
		p[k] = []string{v}
	}
	resp, err := client.PostForm(u, p)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &data)
	return
}
