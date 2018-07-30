package handler

import (
	"errors"
	"fmt"
	"github.com/henrylee2cn/faygo"
	"wallet/model"
)

//获取币种列表
type CoinTypeList struct {
}

func (c *CoinTypeList) Serve(ctx *faygo.Context) error {
	coinTypeModel := new(model.CoinType)
	list, err := coinTypeModel.GetCoinTypeAdmin()
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	faygo.Debug(list[0].Coin_char)
	faygo.Debug(list[0].Coin_price)
	return jsonReturn(ctx, 200, list)
}

//根据币种类型获取用户的钱包账户信息
type AccountWallet struct {
	Uid      string `param:"<in:formData><required><name:uid>"`
	CoinType int    `param:"<in:formData><required><name:cointype>"`
	Offset   int    `param:"<in:formData><required>"`
	Limit    int    `param:"<in:formData><required>"`
	Time     int64  `param:"<in:formData><required>"`
	Sign     string `param:"<in:formData><required>"`
}

func (a *AccountWallet) Serve(ctx *faygo.Context) error {
	err := a.Check(ctx)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	//获取该币种的钱包信息
	coin := model.NewCoin(a.CoinType)
	if coin == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}
	count, err := coin.Count()
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	var list interface{}
	if count > 0 {
		list, err = coin.List(a.Uid, a.Offset, a.Limit)
		if err != nil {
			faygo.Debug(err)
			return jsonReturn(ctx, 0, err.Error())
		}

	}

	return jsonReturn(ctx, 200, list, count)

}

//检查是否合法
func (a *AccountWallet) Check(ctx *faygo.Context) (err error) {
	err = ctx.BindForm(a)
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	b, err := RsaDecrypt([]byte(a.Sign))
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	faygo.Debug(a.Uid, a.CoinType, a.Offset, a.Limit, a.Time)

	if string(b) != Md5(fmt.Sprintf("%v%v%v%v%v", a.Uid, a.CoinType, a.Offset, a.Limit, a.Time)) {
		err = errors.New("非法参数")
	}
	return
}

//用户任务完成(完善用户信息，用户审核通过)添加一条算力提升记录
type AddCalculationInfo struct {
	Uid     string `param:"<in:formData><required><name:uid>"`
	Content string `param:"<in:formData><required><name:content>"`
	Typeid  int    `param:"<in:formData><required><name:type>"`
	Time    int64  `param:"<in:formData><required><name:time>"`
	Num     int    `param:"<in:formData><required><name:num>"`
	Sign    string `param:"<in:formData><required><name:sign>"`
}

func (a *AddCalculationInfo) Serve(ctx *faygo.Context) error {
	err := a.Check(ctx)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	if a.Typeid <= 0 || a.Typeid > 3 {
		return jsonReturn(ctx, 0, "非法参数")
	}
	err = model.DefaultCal.Add(a.Uid, a.Content, a.Num, a.Time, a.Typeid)
	if err != nil {
		faygo.Debug(err)
		return jsonReturn(ctx, 0, err.Error())
	}
	return jsonReturn(ctx, 200, "添加记录成功")
}

//检查是否合法
func (a *AddCalculationInfo) Check(ctx *faygo.Context) (err error) {
	err = ctx.BindForm(a)
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	b, err := RsaDecrypt([]byte(a.Sign))
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	if string(b) != Md5(fmt.Sprintf("%v%v%v", a.Uid, a.Content, a.Time)) {
		err = errors.New("非法参数")
	}
	return
}

//操作币种(禁用，启用)
type CoinAction struct {
	Typeid int    `param:"<in:formData><required><name:typeid>"`
	Status int    `param:"<in:formData><required><name:status>"`
	Time   int64  `param:"<in:formData><required><name:time>"`
	Sign   string `param:"<in:formData><required><name:sign>"`
}

func (c *CoinAction) Serve(ctx *faygo.Context) error {
	if err := c.Check(ctx); err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	coinModel := new(model.CoinType)
	err := coinModel.Action(c.Typeid, c.Status)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	//重置所有的发币程序
	Queue.ReIssue()
	return jsonReturn(ctx, 200, "操作成功")
}

func (c *CoinAction) Check(ctx *faygo.Context) (err error) {
	err = ctx.BindForm(c)
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	b, err := RsaDecrypt([]byte(c.Sign))
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	if string(b) != Md5(fmt.Sprintf("%v%v%v", c.Typeid, c.Status, c.Time)) {
		err = errors.New("非法参数")
	}
	return
}

//添加编辑币种
type CoinSave struct {
	Id             int     `param:"<in:formData><name:id>"`             //id
	Coin_name      string  `param:"<in:formData><name:coin_name>"`      //币种名
	Coin_type      int     `param:"<in:formData><name:coin_type>"`      //币种类型 3比特代币 4以太代币
	Propertyid     int     `param:"<in:formData><name:propertyid>"`     //propertyid
	Addr           string  `param:"<in:formData><name:addr>"`           //合约地址
	Get_price      int     `param:"<in:formData><name:get_price>"`      //是否自动获取价格
	Coin_price     float64 `param:"<in:formData><name:coin_price>"`     //币种价格
	Coin_face      string  `param:"<in:formData><name:coin_face>"`      //币种图标
	Coin_char      string  `param:"<in:formData><name:coin_char>"`      //币种符号
	Coin_unit      string  `param:"<in:formData><name:coin_unit>"`      //币种显示单位
	Coin_unit_base int     `param:"<in:formData><name:coin_unit_base>"` //币种显示单位换算
	Coin_unit_face string  `param:"<in:formData><name:coin_unit_face>"` //币种显示单位图标
	Coin_time      int64   `param:"<in:formData><name:coin_time>"`      //单币发币周期
	Coin_limit     float64 `param:"<in:formData><name:coin_limit>"`     //单次发币最大限制
	Issue_time     int64   `param:"<in:formData><name:issue_time>"`     //发币间隔
	Status         int     `param:"<in:formData><name:status>"`         //币种状态
	Time           int64   `param:"<in:formData><required><name:time>"`
	Sign           string  `param:"<in:formData><required><name:sign>"`
}

func (c *CoinSave) Serve(ctx *faygo.Context) error {
	err := c.Check(ctx)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	if c.Coin_name == "" {
		return jsonReturn(ctx, 0, "币种名称不能为空")
	}

	//验证币种类型
	if c.Coin_type != 3 && c.Coin_type != 4 {
		return jsonReturn(ctx, 0, "请选择币种类型")
	}

	if c.Coin_type == 3 && c.Propertyid == 0 {
		return jsonReturn(ctx, 0, "请输入propertyid")
	}

	if c.Coin_type == 4 && c.Addr == "" {
		return jsonReturn(ctx, 0, "请输入合约地址")
	}

	if c.Get_price == 0 && c.Coin_price == 0 {
		return jsonReturn(ctx, 0, "币种价格不能为空")
	}
	if c.Coin_char == "" {
		return jsonReturn(ctx, 0, "币种符号不能为空")
	}
	if c.Coin_time == 0 {
		return jsonReturn(ctx, 0, "发币周期不能为0")
	}
	if c.Coin_limit == 0 {
		return jsonReturn(ctx, 0, "单次发币不能为0")
	}
	if c.Issue_time < 5 {
		return jsonReturn(ctx, 0, "发币间隔时间不能小于5分钟")
	}
	if c.Status != 0 && c.Status != 1 {
		return jsonReturn(ctx, 0, "非法参数")
	}

	c_type := new(model.CoinType)
	c_type.Name = c.Coin_name
	c_type.ListType = c.Coin_type
	c_type.Addr = c.Addr
	c_type.Propertyid = c.Propertyid
	c_type.Get_price = c.Get_price
	c_type.Coin_price = c.Coin_price
	c_type.Face = c.Coin_face
	c_type.Coin_char = c.Coin_char
	c_type.Unit = c.Coin_unit
	c_type.Base = c.Coin_unit_base
	c_type.Unit_face = c.Coin_unit_face
	c_type.Coin_time = c.Coin_time * 525600
	c_type.Colin_limit = c.Coin_limit
	c_type.Issue_time = c.Issue_time
	c_type.Status = c.Status
	var need_reset_issue bool
	switch c.Id {
	case 0:
		need_reset_issue, err = c_type.Add(c_type)
	default:
		c_type.Id = c.Id
		need_reset_issue, err = c_type.Edit(c_type)
	}

	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	//重置所有的发币程序
	if need_reset_issue {
		Queue.ReIssue()
	}

	return jsonReturn(ctx, 200, "操作成功")
}

func (c *CoinSave) Check(ctx *faygo.Context) (err error) {
	err = ctx.BindForm(c)
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}

	b, err := RsaDecrypt([]byte(c.Sign))
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	if string(b) != Md5(fmt.Sprintf("%v%v%v", c.Coin_char, c.Status, c.Time)) {
		err = errors.New("非法参数")
	}
	return
}

//获取单个币种的信息
type GetCoinTypeById struct {
	Id   int    `param:"<in:formData><required><name:id>"`
	Time int64  `param:"<in:formData><required><name:time>"`
	Sign string `param:"<in:formData><required><name:sign>"`
}

func (c *GetCoinTypeById) Serve(ctx *faygo.Context) error {
	err := c.Check(ctx)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	c_model := new(model.CoinType)
	c_type, err := c_model.GetCoinTypeById(c.Id)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	data := make(map[string]interface{})
	data["name"] = c_type.Name
	data["price"] = c_type.Coin_price
	data["face"] = c_type.Face
	data["char"] = c_type.Coin_char
	data["unit"] = c_type.Unit
	data["unit_face"] = c_type.Unit_face
	data["base"] = c_type.Base
	data["time"] = c_type.Coin_time / 525600
	data["limit"] = c_type.Colin_limit
	data["issue"] = c_type.Issue_time
	data["status"] = c_type.Status
	data["id"] = c_type.Id
	return jsonReturn(ctx, 200, data)
}

func (c *GetCoinTypeById) Check(ctx *faygo.Context) (err error) {
	err = ctx.BindForm(c)
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}

	b, err := RsaDecrypt([]byte(c.Sign))
	if err != nil {
		err = errors.New("参数解析出错")
		return
	}
	if string(b) != Md5(fmt.Sprintf("%v%v", c.Id, c.Time)) {
		err = errors.New("非法参数")
	}
	return
}
