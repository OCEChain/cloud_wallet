package config

import (
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/ini"
)

//获取my.ini配置中的配置
func GetConfig(section, key string) (res *ini.Key) {
	cfg, err := ini.Load(faygo.CONFIG_DIR + "my.ini")
	if err != nil {
		return
	}
	res = cfg.Section(section).Key(key)
	return
}
