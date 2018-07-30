package model

type Version struct {
	Id      int    `xorm:"not null INT(11) pk autoincr"`
	Version string `xorm:"not null default('') char(30) comment('版本号')"`
}
