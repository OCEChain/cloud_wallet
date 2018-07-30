package model

import (
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
)

//用户获取nouceid 的表

type Ids struct {
	Id     int `xorm:"not null INT(11) pk autoincr"`
	Is_use int `xorm:"not null default(0) tinyint(4) comment('是否使用了')"`
}

var DefaultIds = new(Ids)

const (
	IdsTable = "ids"
)

func init() {
	err := xorm.MustDB().Table(IdsTable).Sync2(DefaultIds)
	if err != nil {
		faygo.Error(err.Error())
	}
}
