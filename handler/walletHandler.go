package handler

import (
	"encoding/json"
	"fmt"
	"github.com/henrylee2cn/faygo"
	"math/big"
	"strconv"
	"strings"
	"wallet/config"
	"wallet/model"
	"wallet/redis"
)

//登陆到钱包上，返回当前各个币种的各项信息收益
type Wallet struct {
	Token       string `param:"<in:formData><required><name:token><desc:用户登陆后获取的token>"`
	Device_code string `param:"<in:formData><required><name:device_code><desc:设备码>"`
}

func (w *Wallet) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("返回当前各个币种的各项信息收益接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(w)
	if err != nil {
		return jsonReturn(ctx, 0, "参数解析出错")
	}

	if w.Token == "" || w.Device_code == "" {
		return jsonReturn(ctx, 0, "参数不能为空")
	}
	return_data := make(map[string]interface{})

	//通过用户token获取用户信息
	user, err := NewUser(w.Token)
	if err == LoginExpire {
		return jsonReturn(ctx, 1, err.Error())
	}
	if err != nil {
		return jsonReturn(ctx, 2, err.Error())
	}
	uid := user.GetUidByToken()
	res := uid
	res_data, err := redis.New().Get(res)
	if err == nil {
		err = json.Unmarshal([]byte(res_data), &return_data)
		if err != nil {
			return jsonReturn(ctx, 0, "服务器出错")
		}
		return jsonReturn(ctx, 200, return_data)
	}
	//获取设备信息
	device, err := model.NewDeviceByCode(w.Device_code)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}

	//判断是否是上一次在该设备上登陆的，如果不是，则绑定现在的uid，并且将之前的那个uid对应的挖矿程序结束掉
	if device.Uid != uid {
		err = device.Bind(w.Device_code, uid)
		if err != nil {
			faygo.Debug(uid)
			return jsonReturn(ctx, 0, err.Error())
		}
		//停止掉之前的uid的挖矿程序
		Queue.Stop(device.Uid)
		//移除之前的算力
		ALlCal.DelCal(device.Uid)
	}
	//获取各种币种信息
	//获取各种币

	coinTypeList := model.AllCoinType.GetAllInfoList()
	dataList := []interface{}{}
	for _, v := range coinTypeList {
		if v == nil {
			continue
		}
		//获取用户的各种币的信息
		coin := model.NewCoin(v.Id)
		err = coin.GetInfo(uid)
		if err != nil {
			return jsonReturn(ctx, 0, err.Error())
		}
		data_v := *v
		if v.Unit_face != "" {
			data_v.Unit_face = config.GetConfig("admin_url", "url").String() + data_v.Unit_face

		}
		if v.Face != "" {
			data_v.Face = config.GetConfig("admin_url", "url").String() + data_v.Face
		}

		data := make(map[string]interface{})
		data["coin_info"] = data_v                   //币种信息
		data["profit"] = coin.Profit                 //累计收益
		data["balance"] = coin.Balance               //余额(已经领取收益,累计收益-未领取的收益)
		data["receive_profit"] = coin.Receive_profit //可领取的收益
		dataList = append(dataList, data)
	}
	return_data["list"] = dataList

	//让当前用户开启发币
	Queue.Issue(uid)
	//重置算力
	//当前的算力为
	var cal float64 = 15
	if user.CheckHasInfo() {
		cal = cal + 10
	}

	if user.CheckIsInvite() {
		cal = cal + 20
	}

	cal = cal + 10*user.GetInviteNum()
	faygo.Debug(cal)
	ALlCal.SetCal(uid, cal)
	//算力
	return_data["calculation"] = cal
	b, err := json.Marshal(return_data)
	if err != nil {
		return jsonReturn(ctx, 0, "服务器出错")
	}
	redis.New().Set(res, string(b), 300)
	return jsonReturn(ctx, 200, return_data)
}

func (w *Wallet) Doc() faygo.Doc {
	err := return_jonData(0, "获取失败")
	success := return_jonData(200, "")
	return_param := []interface{}{err, success}

	return faygo.Doc{
		Note:   "获取钱包收益的接口",
		Return: return_param,
	}
}

//获取账本
type Book struct {
	Token    string `param:"<in:formData><required><name:token><desc:用户登陆后获取的token>"`
	CoinType string `param:"<in:formData><required><name:cointype><desc:转出的币种类型，btc ,eth >"`
	Page     int    `param:"<in:formData><required><name:page><desc:页码>"`
}

func (b *Book) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("获取账本接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(b)
	if err != nil {
		return jsonReturn(ctx, 0, "参数解析出错")
	}

	if b.Token == "" {
		return jsonReturn(ctx, 0, "token不能为空")
	}

	if b.Page <= 0 {
		b.Page = 1
	}

	//通过用户token获取用户信息
	user, err := NewUser(b.Token)
	if err == LoginExpire {
		return jsonReturn(ctx, 1, err.Error())
	}
	if err != nil {
		return jsonReturn(ctx, 2, err.Error())
	}
	uid := user.GetUidByToken()

	//设置每页最多有10条记录
	limit := 10
	data := make(map[string]interface{})
	data["page"] = b.Page
	data["limit"] = limit

	//coinType := model.AllCoinType.GetOneCoinInfoByTypeid(b.CoinType)
	coinType := model.AllCoinType.GetOneCoinInfoByChar(b.CoinType)
	if coinType == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}
	coinLog := model.NewCoinLog(coinType.Id)
	count, err := coinLog.Count(uid)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	data["count"] = count
	data["list"] = []interface{}{}
	if count > 0 {
		offset := (b.Page - 1) * limit
		list, err := coinLog.List(uid, offset, limit)
		if err != nil {
			return jsonReturn(ctx, 0, err.Error())
		}
		if len(list) > 0 {
			data["list"] = list
		}
	}

	return jsonReturn(ctx, 200, data)
}

func (b *Book) Doc() faygo.Doc {
	err := return_jonData(0, "获取失败")
	success := return_jonData(200, "")
	return_param := []interface{}{err, success}

	return faygo.Doc{
		Note:   "获取账本信息的接口",
		Return: return_param,
	}
}

//领取
type Receive_profit struct {
	Token    string `param:"<in:formData><required><name:token><desc:用户登陆后获取的token>"`
	CoinType string `param:"<in:formData><required><name:cointype><desc:转出的币种类型，btc ,eth >"`
}

func (r *Receive_profit) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("领取接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(r)
	if err != nil {
		return jsonReturn(ctx, 0, "参数解析出错")
	}
	if r.Token == "" {
		return jsonReturn(ctx, 0, "token不能为空")
	}

	//通过用户token获取用户信息
	user, err := NewUser(r.Token)
	if err == LoginExpire {
		return jsonReturn(ctx, 1, err.Error())
	}
	if err != nil {
		return jsonReturn(ctx, 2, err.Error())
	}
	uid := user.GetUidByToken()
	c_type := model.AllCoinType.GetOneCoinInfoByChar(r.CoinType)
	if c_type == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}

	//清空
	coinType := model.NewCoin(c_type.Id)
	if coinType == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}
	err = coinType.Receive(uid)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	res := uid
	redis.New().Del(res)
	return jsonReturn(ctx, 200, "领取成功")
}

func (r *Receive_profit) Doc() faygo.Doc {
	err := return_jonData(0, "获取失败")
	success := return_jonData(200, "")
	return_param := []interface{}{err, success}

	return faygo.Doc{
		Note:   "领取收益的接口",
		Return: return_param,
	}
}

//转出
type Transfer struct {
	//Tradepwd string  `param:"<in:formData><required><name:tradepwd><desc:用户的交易密码>"`
	Num      float64 `param:"<in:formData><required><name:num><desc:转出的数额>"`
	CoinType string  `param:"<in:formData><required><name:cointype><desc:转出的币种类型，btc ,eth>"`
	Addr     string  `param:"<in:formData><required><name:addr><desc:转出的地址>"`
	Token    string  `param:"<in:formData><required><name:token><desc:用户登陆后获取的token>"`
	FeeType  int     `param:"<in:formData><name:feetype><desc:选择的手续费类型,只有转比特币的时候有用，1表示fastestFee 2表示halfHourFee 3表示hourFee>"`
}

func (t *Transfer) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("转出接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(t)
	if err != nil {
		return jsonReturn(ctx, 0, "参数解析出错")
	}
	if t.Token == "" {
		return jsonReturn(ctx, 0, "token不能为空")
	}
	//if !CheckTradepwd(t.Tradepwd) {
	//	return jsonReturn(ctx, 0, "请输入六位数字交易密码")
	//}

	//通过用户token获取用户信息
	user, err := NewUser(t.Token)
	if err == LoginExpire {
		return jsonReturn(ctx, 1, err.Error())
	}
	if err != nil {
		return jsonReturn(ctx, 2, err.Error())
	}
	uid := user.GetUidByToken()
	//检验交易密码是否正确
	//ok, err := user.CheckTradepwd(t.Tradepwd)
	//if err != nil {
	//	return jsonReturn(ctx, 0, err.Error())
	//}
	//
	//if !ok {
	//	return jsonReturn(ctx, 0, "交易密码不正确")
	//}
	//获取币种
	coinType := model.AllCoinType.GetOneCoinInfoByChar(t.CoinType)
	if coinType == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}

	if coinType.ListType == 2 || coinType.ListType == 4 {
		t.Addr = strings.ToLower(t.Addr)
		//判断地址是否正确
		if !strings.HasPrefix(t.Addr, "0x") {
			return jsonReturn(ctx, 0, "请输入正确格式的转账地址")
		}
	}

	//检查转出是否足够
	coin := model.NewCoin(coinType.Id)
	if coin == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}
	err = coin.GetInfo(uid)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	if coin.Balance < t.Num {
		return jsonReturn(ctx, 0, "余额不足")
	}

	if coinType.ListType == 0 || coinType.Transfer_status == 0 {
		return jsonReturn(ctx, 0, "当前币种暂时不可以转账")
	}
	var fee float64
	//获取手续费
	switch coinType.ListType {
	case 1, 3:
		//比特类型
		res := coinFee.Get(1)
		data := make(map[string]interface{})
		err = json.Unmarshal([]byte(res), &data)
		if err != nil {
			return jsonReturn(ctx, 0, "服务器出错")
		}

		switch t.FeeType {
		case 1:
			fee, _ = strconv.ParseFloat(data["fastestFee"].(string), 64)
		case 2:
			fee, _ = strconv.ParseFloat(data["halfHourFee"].(string), 64)
		case 3:
			fee, _ = strconv.ParseFloat(data["hourFee"].(string), 64)
		default:
			return jsonReturn(ctx, 0, "非法参数")
		}
	case 2, 4:
		//eth
		res := coinFee.Get(2)

		gas, err := strconv.Atoi(res)
		if err != nil {
			return jsonReturn(ctx, 0, "服务器出错")
		}
		//faygo.Debug(gas / 1000000000000000)

		gas = gas * 21000
		var r int64
		res_float := big.NewRat(r, 1)
		ze_float := big.NewRat(1000000000000000000, 1)
		gas_float := big.NewRat(int64(gas), 1)

		res_float = res_float.Quo(gas_float, ze_float)
		fee, _ = res_float.Float64()
	default:
		return jsonReturn(ctx, 0, "获取不到手续费")
	}

	if fee >= t.Num {
		return jsonReturn(ctx, 0, "转出的数额不能比手续费少")
	}

	//添加交易记录
	trade_model := model.NewTrade(coinType.Id)
	err = trade_model.Add(coinType.Id, coin.Balance, fee, uid, t.Addr, t.Num)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	return jsonReturn(ctx, 0, "转出成功")
}

func (t *Transfer) Doc() faygo.Doc {
	err := return_jonData(0, "获取失败")
	success := return_jonData(200, "")
	return_param := []interface{}{err, success}

	return faygo.Doc{
		Note:   "转出接口",
		Return: return_param,
	}
}

//获取手续费
type GetPoundage struct {
	CoinType string `param:"<in:formData><required><name:cointype><desc:币种类型 btc eth>"`
}

func (g *GetPoundage) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("获取手续费接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(g)
	if err != nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}

	//获取币种
	coinType := model.AllCoinType.GetOneCoinInfoByChar(g.CoinType)
	if coinType == nil {
		return jsonReturn(ctx, 0, "不存在的币种类型")
	}
	switch coinType.ListType {
	case 1, 3:
		//比特类型
		res := coinFee.Get(1)
		faygo.Debug(res)
		fee := make(map[string]interface{})
		err = json.Unmarshal([]byte(res), &fee)
		if err != nil {
			faygo.Debug(err)
			return jsonReturn(ctx, 0, "服务器出错")
		}

		return jsonReturn(ctx, 200, fee)
	case 2, 4:
		//eth
		res := coinFee.Get(2)

		gas, err := strconv.Atoi(res)
		if err != nil {
			return jsonReturn(ctx, 0, "服务器出错")
		}
		//faygo.Debug(gas / 1000000000000000)

		gas = gas * 21000
		var r int64
		res_float := big.NewRat(r, 1)
		ze_float := big.NewRat(1000000000000000000, 1)
		gas_float := big.NewRat(int64(gas), 1)

		res_float = res_float.Quo(gas_float, ze_float)
		faygo.Debug(res_float.FloatString(7))
		faygo.Debug(res_float.Float64())
		return jsonReturn(ctx, 200, res_float.FloatString(7))
	default:
		return jsonReturn(ctx, 0, "获取不到手续费")
	}
}

//意见反馈
type Feedback struct {
	Token   string `param:"<in:formData><required><name:token><desc:用户登陆后获取的token>"`
	Content string `param:"<in:formData><required><name:content><desc:内容>"`
	Contact string `param:"<in:formData><name:contact><desc:联系方式>"`
}

func (f *Feedback) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("意见反馈接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(f)
	if err != nil {
		return jsonReturn(ctx, 0, "参数解析出错")
	}
	if f.Token == "" {
		return jsonReturn(ctx, 0, "token不能为空")
	}

	if f.Content == "" {
		return jsonReturn(ctx, 0, "不能为空")
	}

	//通过用户token获取用户信息
	user, err := NewUser(f.Token)
	if err == LoginExpire {
		return jsonReturn(ctx, 1, err.Error())
	}
	if err != nil {
		return jsonReturn(ctx, 2, err.Error())
	}

	uid := user.GetUidByToken()
	err = model.DefaultFeedback.Add(uid, f.Content, f.Contact)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	return jsonReturn(ctx, 200, "反馈成功")
}

//返回算力提升记录
type PowerRecord struct {
	Token string `param:"<in:formData><required><name:token><desc:用户登陆后获取的token>"`
	Page  int    `param:"<in:formData><required><name:page><desc:页码>"`
}

func (p *PowerRecord) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("返回算力提升记录接口出现一个意外错误，错误信息为", err)
		}
	}()
	err := ctx.BindForm(p)
	if err != nil {
		return jsonReturn(ctx, 0, "参数解析出错")
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	//获取uid
	//通过用户token获取用户信息
	user, err := NewUser(p.Token)
	if err == LoginExpire {
		return jsonReturn(ctx, 1, err.Error())
	}
	if err != nil {
		return jsonReturn(ctx, 2, err.Error())
	}

	uid := user.GetUidByToken()

	count, err := model.DefaultCal.Count(uid)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	limit := 10
	var list interface{}
	if count > 0 {
		offset := (p.Page - 1) * limit
		list, err = model.DefaultCal.List(uid, offset, limit)
		if err != nil {
			return jsonReturn(ctx, 0, err.Error())
		}
	}
	data := make(map[string]interface{})
	data["list"] = list
	data["count"] = count
	data["limit"] = limit
	data["page"] = p.Page
	return jsonReturn(ctx, 200, data)

}

//获取各个币种的价格
type GetPrice struct {
	Coin string `param:"<in:formData><required><name:cointype><desc:币种>"`
}

func (g *GetPrice) Serve(ctx *faygo.Context) error {
	defer func() {
		if err := recover(); err != nil {
			faygo.Info("获取币种价格接口出现一个意外错误，错误信息为", err)
		}
	}()
	//获取缓存中的数据
	data, err := redis.New().Get(fmt.Sprintf("price_$v", g.Coin))
	if err != nil {
		data, err = model.GetPrice(g.Coin)
		if err != nil {
			return jsonReturn(ctx, 0, err.Error())
		}
		//写入到缓存中
		redis.New().Set(fmt.Sprintf("price_$v", g.Coin), data, 180)
	}
	return jsonReturn(ctx, 200, data)

}

//添加设备码的接口
type AddDeviceCode struct {
	Code         string `param:"<in:formData><required><name:code>"`
	Another_code string `param:"<in:formData><required><name:another_code>"`
}

func (a *AddDeviceCode) Serve(ctx *faygo.Context) error {
	if err := ctx.BindForm(a); err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	device_model := new(model.Device)
	err := device_model.Add(a.Code, a.Another_code)
	if err != nil {
		return jsonReturn(ctx, 0, err.Error())
	}
	return jsonReturn(ctx, 200, "添加成功")
}
