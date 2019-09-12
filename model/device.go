package model

import (
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
)

//设备
type Device struct {
	Id           int    `xorm:"not null INT(11) pk autoincr"`
	Code         string `xorm:"not null default('') varchar(50) comment('机器码')"`
	Another_code string `xorm:"not null default('') varchar(50) comment('副卡')"`
	Uid          string `xorm:"not null default('') char(20) comment('绑定用户uid')"`
}

const (
	DeviceTable = "device"
)

var (
	NotJGW = errors.New("不是JGW区块链手机")
)

func init() {
	err := xorm.MustDB().Table(DeviceTable).Sync2(new(Device))
	if err != nil {
		faygo.Error(err.Error())
	}
}

//通过设备获取设备信息
func NewDeviceByCode(device_code string) (device *Device, err error) {
	engine := xorm.MustDB()
	device = new(Device)
	has, err := engine.Where("code=?", device_code).Or("another_code=?", device_code).Get(device)
	if err != nil {
		err = SystemFail
		return
	}
	if !has {
		err = NotJGW
	}
	return
}

func (d *Device) Bind(device_code, uid string) (err error) {
	engine := xorm.MustDB()
	//清空之前用户登陆过的其他设备的uid
	device := new(Device)
	device.Uid = ""
	_, err = engine.Table(DeviceTable).Where("uid=?", uid).Cols("uid").Update(device)
	if err != nil {
		err = SystemFail
		return
	}
	//将当前设备的uid设置成当前uid
	device.Uid = uid
	n, err := engine.Table(DeviceTable).Where("code=?", device_code).Or("another_code=?", device_code).Cols("uid").Update(device)
	if err != nil || n == 0 {
		err = SystemFail
	}
	return
}

//添加一个设备
func (d *Device) Add(code string, another_code string) (err error) {
	engine := xorm.MustDB()
	device := new(Device)
	device.Code = code
	device.Another_code = another_code
	_, err = engine.Insert(device)
	return
}
