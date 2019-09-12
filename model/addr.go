package model

import (
	"github.com/go-errors/errors"
	x "github.com/go-xorm/xorm"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
)

//管理全局交易序列号
type AddrSerialNum struct {
	Id         int `xorm:"not null INT(11) pk autoincr"`
	Serial_num int `xorm:"not null default(0) int(11) comment('交易序列号')"`
	Last_time  int `xorm:"not null default(0) int(11) comment('最后一次增加交易id的时间')"`
}

var DefaultAddrSerialNum = new(AddrSerialNum)

const (
	AddrSerialNumTable = "addr_serial_num"
)

func init() {
	err := xorm.MustDB().Table(AddrSerialNumTable).Sync2(DefaultAddrSerialNum)
	if err != nil {
		faygo.Error(err.Error())
	}
}

//获取一条记录
func (a *AddrSerialNum) GetSerialNum(sess *x.Session) (id int, err error) {

	ids := new(Ids)
	has, err := sess.Where("is_use=?", 0).Get(ids)
	if err != nil {
		return
	}

	//如果没有,曾从新生成一个，并且累加
	if !has {
		addrSerialNum := new(AddrSerialNum)
		b, err := sess.Table(AddrSerialNumTable).Where("id=?", 1).Get(addrSerialNum)
		if err != nil {
			return 0, err
		}
		if !b {
			addrSerialNum.Serial_num = 1
			addrSerialNum.Id = 1
			//不存在则创建一条记录
			n, err := sess.Insert(addrSerialNum)
			if err != nil {
				return 0, err
			}
			if n == 0 {
				err = errors.New("创建失败")
				return 0, err
			}
			addrSerialNum.Id = 1
		}
		ids = new(Ids)
		ids.Id = addrSerialNum.Serial_num
		ids.Is_use = 1 //表明已经使用了该id
		n, err := sess.Insert(ids)
		if err != nil || n == 0 {
			err = SystemFail
			return 0, err
		}
		//全局交易id累加
		res, err := sess.Exec("update addr_serial_num set serial_num=serial_num+1")
		if err != nil {
			err = SystemFail
			return 0, err
		}
		n, err = res.RowsAffected()
		if err != nil || n == 0 {
			return 0, err
		}

	} else {
		ids.Is_use = 1
		_, err := sess.Where("id=?", ids.Id).Update(ids)
		if err != nil {
			return 0, err
		}
	}
	id = ids.Id
	return
}

func (a *AddrSerialNum) Get() (addrSerialNum *AddrSerialNum, err error) {
	engine := xorm.MustDB()
	sess := engine.NewSession()
	defer sess.Close()
	err = sess.Begin()
	if err != nil {
		err = SystemFail
		return
	}

	return
}

func (a *AddrSerialNum) create() (err error) {
	engine := xorm.MustDB()
	a.Serial_num = 1
	a.Id = 1
	n, err := engine.Insert(a)
	if err != nil {
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("创建失败")
	}
	return
}
