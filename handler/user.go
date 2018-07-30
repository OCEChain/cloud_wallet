package handler

import (
	"github.com/go-errors/errors"
	"time"
	"wallet/config"
	"wallet/model"
)

type user struct {
	data map[string]interface{}
}

var LoginExpire = errors.New("用户登陆失效")

func NewUser(token string) (u *user, err error) {
	u = new(user)
	url := config.GetConfig("user", "url").String() + "/getinfo"
	param := make(map[string]string)
	param["token"] = token
	data, err := curl_post(url, param, time.Second*5)
	if err != nil {
		err = model.SystemFail
		return
	}
	if data.Code == 1 {
		err = LoginExpire
		return
	}
	if data.Code != 200 {
		err = errors.New(data.Data)
		return
	}
	userInfo, ok := data.Data.(map[string]interface{})
	if !ok {
		err = errors.New("解析出错")
		return
	}
	u.data = userInfo
	return
}

//通过token获取用户uid
func (u *user) GetUidByToken() (uid string) {
	userInfo := u.data
	uid = userInfo["Uid"].(string)
	return
}

//检验用户的交易密码是否正确
func (u *user) CheckTradepwd(tradepwd string) (ok bool, err error) {
	//暂时省略
	//假设交易密码正确
	ok = true
	return
}

//检查用户信息是否有设置
func (u *user) CheckHasInfo() (b bool) {
	nickname := u.data["Nickname"].(string)
	face := u.data["Face"].(string)
	if nickname != "" && face != "" {
		b = true
	}
	return
}

//检查是否已经身份认证
func (u *user) CheckIsInvite() (b bool) {
	status := u.data["Audit_status"].(float64)
	if status == 2 {
		b = true
	}
	return
}

//获取用户的邀请好友数
func (u *user) GetInviteNum() float64 {
	invite_num := u.data["Invite_man_num"].(float64)
	return invite_num
}
