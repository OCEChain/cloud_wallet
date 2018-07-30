package model

import (
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
)

//算力提升记录
type Calculation struct {
	Id      int    `xorm:"not null INT(11) pk autoincr"`
	Uid     string `xorm:"not null default('') varchar(50) comment('uid')" json:"-"`
	Content string `xorm:"not null default('') varchar(255) comment('内容')"`
	Typeid  int    `xorm:"not null default(0) tinyint(4) comment('1表示完善信息 2表示审核通过 3表示邀请好友')"`
	Num     int    `xorm:"not null default(0) int(11) comment('提升的算力值')"`
	Time    int64  `xorm:"not null default(0) int(11) comment('时间')"`
}

const CalTable = "calculation"

var DefaultCal = new(Calculation)

func init() {
	err := xorm.MustDB().Table(CalTable).Sync2(DefaultCal)
	if err != nil {
		faygo.Error(err.Error())
	}
}

//添加一条记录
func (c *Calculation) Add(uid, content string, num int, t int64, typeid int) (err error) {
	engine := xorm.MustDB()
	ca := new(Calculation)
	ca.Uid = uid
	ca.Content = content
	ca.Num = num
	ca.Time = t
	ca.Typeid = typeid
	n, err := engine.Insert(ca)
	if err != nil {
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("添加记录失败")
	}
	return
}

//获取算力提升记录
func (c *Calculation) List(uid string, offset, limit int) (list []*Calculation, err error) {
	engine := xorm.MustDB()
	rows, err := engine.Desc("id").Where("uid=?", uid).Rows(c)
	if err != nil {
		err = SystemFail
		return
	}
	defer rows.Close()
	for rows.Next() {
		cal := new(Calculation)
		err = rows.Scan(cal)
		if err != nil {
			err = SystemFail
			return
		}
		list = append(list, cal)
	}
	return
}

func (c *Calculation) Count(uid string) (count int64, err error) {
	engine := xorm.MustDB()
	count, err = engine.Table(CalTable).Where("uid=?", uid).Count()
	if err != nil {
		err = SystemFail
	}
	return
}
