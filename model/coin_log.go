package model

import (
	"errors"
	x "github.com/go-xorm/xorm"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"time"
)

type coinLog struct {
	Id         int     `xorm:"not null INT(11) pk autoincr" json:"id"`
	Uid        string  `xorm:"not null index default('') char(20) comment('用户uid')" json:"-"`
	Num        float64 `xorm:"not null default(0.00000000) decimal(18,8) comment('数量')" json:"num"`
	Typeid     int     `xorm:"not null index default(0) tinyint(4) comment('类型，1:领取, 2:转出 3:返还')" json:"type"`
	Balance    float64 `xorm:"not null default(0.00000000) decimal(18,8) comment('操作完后的余额')" json:"balance"`
	Time       int64   `xorm:"not null default(0) int(11) comment('操作时间')" json:"time"`
	Addr       string  `xorm:"not null default('') varchar(255) comment('转账操作操做的地址')" json:"addr"`
	tableName  string  `xorm:"-"`
	coinTypeid int     `xorm:"-"`
}

//构造函数(根据币种类型ID来获取表名，构建各种查询模型，通过)
func NewCoinLog(typeid int, find_tab ...bool) (c *coinLog) {
	c = new(coinLog)
	c.coinTypeid = typeid
	var coinType *CoinType
	if len(find_tab) == 0 {
		coinType = AllCoinType.GetOneCoinInfoByTypeid(typeid)
		if coinType == nil {
			return nil
		}
	} else {
		var err error
		coinType, err = new(CoinType).GetCoinTypeById(typeid)
		if err != nil {
			return nil
		}
	}

	c.setTableName(coinType.Coin_char + "_log")
	return
}

//创建表
func (c *coinLog) createTab() {
	err := xorm.MustDB().Table(c.tableName).Sync2(new(coinLog))
	if err != nil {
		faygo.Error(err.Error())
	}
}

func (c *coinLog) setTableName(tableName string) (co *coinLog) {
	c.tableName = tableName
	return c
}

//增加一条记录
func (c *coinLog) AddLog(sess *x.Session, uid string, typeid int, num, balance float64, addr string) (err error) {
	c.Uid = uid
	c.Num = num
	c.Typeid = typeid
	c.Balance = balance
	c.Time = time.Now().Unix()
	c.Addr = addr
	n, err := sess.Table(c.tableName).Insert(c)
	if err != nil {
		faygo.Debug(err)
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("添加失败")
	}
	return
}

//获取记录
func (c *coinLog) List(uid string, offset, limit int) (list []*coinLog, err error) {
	engine := xorm.MustDB()
	rows, err := engine.Table(c.tableName).Where("uid=?", uid).Desc("id").Limit(limit, offset).Rows(c)
	if err != nil {
		err = SystemFail
		return
	}
	defer rows.Close()
	for rows.Next() {
		btclog := new(coinLog)
		err = rows.Scan(btclog)
		if err != nil {
			err = SystemFail
			return
		}
		list = append(list, btclog)
	}
	return
}

//获取总记录数
func (c *coinLog) Count(uid string) (count int64, err error) {
	engine := xorm.MustDB()
	count, err = engine.Table(c.tableName).Where("uid=?", uid).Count(c)
	return
}
