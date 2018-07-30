package model

import (
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"time"
)

type Feedback struct {
	Id      int    `xorm:"not null INT(11) pk autoincr"`
	Uid     string `xorm:"not null default('') char(20) comment('uid')"`
	Content string `xorm:"not null TEXT comment('内容')"`
	Contact string `xorm:"not null default('') comment('联系方式')"`
	Time    int64  `xorm:"not null default(0) int(11) comment('时间')"`
}

var DefaultFeedback = new(Feedback)

const (
	FeedbackTable = "feedback"
)

func init() {
	err := xorm.MustDB().Table(FeedbackTable).Sync2(DefaultFeedback)
	if err != nil {
		faygo.Error(err.Error())
	}
}

func (f *Feedback) Add(uid, content, contact string) (err error) {
	engine := xorm.MustDB()
	feedback := new(Feedback)
	feedback.Uid = uid
	feedback.Content = content
	feedback.Contact = contact
	feedback.Time = time.Now().Unix()
	n, err := engine.Insert(feedback)
	if err != nil {
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("反馈失败")
	}
	return

}
