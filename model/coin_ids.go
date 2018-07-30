package model

import (
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"strings"
)

//存放获取价格接口的币种的id
type CoinIds struct {
	Id        int    `xorm:"not null INT(11) pk autoincr"`
	Coin_char string `xorm:"not null default('') varchar(30) comment('币种符号，如ETH')"`
}

var DefaultCoinIds = new(CoinIds)

const (
	CoinIdsTable = "coin_ids"
)

func init() {
	err := xorm.MustDB().Table(CoinIdsTable).Sync2(DefaultCoinIds)
	if err != nil {
		faygo.Error(err.Error())
	}
}

func (c *CoinIds) Add(id int, char string) (err error) {
	engine := xorm.MustDB()
	coin_ids := new(CoinIds)
	coin_ids.Id = id
	coin_ids.Coin_char = char
	n, err := engine.Insert(coin_ids)
	if err != nil {
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("插入失败")
	}
	return
}

func (c *CoinIds) GetIdByChar(char string) (id int, err error) {
	char = strings.ToUpper(char)
	engine := xorm.MustDB()
	coin_ids := new(CoinIds)
	has, err := engine.Where("coin_char=?", char).Get(coin_ids)
	if err != nil {
		err = SystemFail
		return
	}
	if !has {
		err = errors.New("获取不到当前币种的id")
		return
	}
	id = coin_ids.Id
	return
}
