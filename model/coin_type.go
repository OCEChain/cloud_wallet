package model

import (
	"github.com/go-errors/errors"
	"github.com/henrylee2cn/faygo"
	"github.com/henrylee2cn/faygo/ext/db/xorm"
	"time"
)

//币种类型
type CoinType struct {
	Id              int     `xorm:"not null INT(11) pk autoincr" json:"id"`
	Name            string  `xorm:"not null default('') varchar(50) comment('币种名称，如比特币')" json:"name"`
	Coin_char       string  `xorm:"not null default('') varchar(20) comment('币种代号，如 btc, eth')" json:"char"`
	Face            string  `xorm:"not null default('') varchar(255) comment('币种图标')" json:"face"`
	Unit            string  `xorm:"not null default('') varchar(20) comment('单位')" json:"unit"`
	Unit_face       string  `xorm:"not null default('') varchar(255) comment('单位的图片')" json:"unit_face"`
	Base            int     `xorm:"not null default(0) int(11) comment('如果使用自定义的单位，则乘以这个数进行展示')"`
	Status          int     `xorm:"not null default(0) tinyint(4) comment('状态，1表示可用，0表示禁用')" json:"-"`
	Addr            string  `xorm:"not null default('') varchar(50) comment('合约地址')"`
	Propertyid      int     `xorm:"not null default(0) int(11) comment('propertyId')"`
	Transfer_status int     `xorm:"not null default(0) tinyint(4) comment('转账状态 1开启转账 2未开启转账')"`
	ListType        int     `xorm:"not null default(0) tinyint(4) comment('上链的类型1表示比特币 2以太币 3比特代币 4以太代币')" json:"-"`
	Get_price       int     `xorm:"not null default(0) int(11) comment('是否自动获取价格')" json:"-"`
	Coin_price      float64 `xorm:"not null default(0.00) decimal(10,2) comment('币种价格')" json:"-"`
	Coin_time       int64   `xorm:"not null default(0) int(11) comment('期限,发币期限，单位分钟')" json:"-"`
	Colin_limit     float64 `xorm:"not null default(0.00000000) decimal(10,8) comment('没次分发的限额')" json:"-"`
	Issue_time      int64   `xorm:"not null default(0) int(11) comment('发币间隔时间')" json:"-"`
	Update_time     int64   `xorm:"not null default(0) int(11) comment('修改时间')" json:"-"`
}

const (
	CoinTypeTABLE = "coin_type"
)

type CoinTypeAdmin struct {
	*CoinType
	Coin_price  float64
	Coin_time   int64
	Colin_limit float64
	Issue_time  int64
	Status      int
}

//获取所有的币种类型（后台管理用的）

//获取所有的币种类型
func (c *CoinType) GetCoinTypeAdmin() (typeListAdmin []*CoinTypeAdmin, err error) {
	engine := xorm.MustDB()
	rows, err := engine.Rows(c)
	if err != nil {
		err = SystemFail
		return
	}
	defer rows.Close()

	for rows.Next() {
		coin_type := new(CoinType)
		err = rows.Scan(coin_type)
		if err != nil {
			err = SystemFail
			return
		}
		co := new(CoinTypeAdmin)
		co.CoinType = coin_type
		co.Coin_price = coin_type.Coin_price
		co.Coin_time = coin_type.Coin_time
		co.Colin_limit = coin_type.Colin_limit
		co.Issue_time = coin_type.Issue_time
		co.Status = coin_type.Status
		typeListAdmin = append(typeListAdmin, co)
	}
	return
}

//获取所有的币种类型
func (c *CoinType) GetCoinType() (list map[string]*CoinType, listById map[int]*CoinType, TypeList []*CoinType, typeListAdmin []*CoinTypeAdmin, err error) {
	engine := xorm.MustDB()
	rows, err := engine.Where("status=?", 1).Rows(c)
	if err != nil {
		err = SystemFail
		return
	}
	defer rows.Close()
	list = make(map[string]*CoinType)
	listById = make(map[int]*CoinType)
	for rows.Next() {
		coin_type := new(CoinType)
		err = rows.Scan(coin_type)
		if err != nil {
			faygo.Debug(err)
			err = SystemFail
			return
		}

		list[coin_type.Coin_char] = coin_type
		listById[coin_type.Id] = coin_type
		TypeList = append(TypeList, coin_type)
		co := new(CoinTypeAdmin)
		co.CoinType = coin_type
		co.Coin_price = coin_type.Coin_price
		co.Coin_time = coin_type.Coin_time
		co.Colin_limit = coin_type.Colin_limit
		co.Issue_time = coin_type.Issue_time
		co.Status = coin_type.Status
		typeListAdmin = append(typeListAdmin, co)
	}

	return
}

//操作币种（禁用，启用）
func (c *CoinType) Action(typeid int, status int) (err error) {
	if status < 0 || status > 1 {
		err = errors.New("错误的类型")
		return
	}

	//判断是否已经有禁用或者启用的操作在执行
	if AllCoinType.GetResetStatus() {
		err = errors.New("当前系统正处于重置币种中，请稍后再试")
		return
	}
	AllCoinType.SetReset(true) //设置成重置币种的状态
	engine := xorm.MustDB()
	c_type := new(CoinType)
	c_type.Status = status
	n, err := engine.Where("id=?", typeid).Cols("status").Update(c_type)
	if err != nil {
		AllCoinType.SetReset(false) //设置成重置币种的状态
		err = SystemFail
		return
	}
	if n == 0 {
		AllCoinType.SetReset(false) //设置成重置币种的状态
		err = errors.New("操作失败")
		return
	}
	//重新初始化币种信息
	err = AllCoinType.Init()
	if err != nil {
		AllCoinType.SetReset(false) //设置成重置币种的状态
		return
	}
	return
}

//添加币种
func (c *CoinType) Add(c_type *CoinType) (need_reset_issue bool, err error) {
	engine := xorm.MustDB()
	//查询是否已经存在相同币种符号或者名称的币种
	t := new(CoinType)
	has, err := engine.Where("name=? or coin_char=?", c_type.Name, c_type.Coin_char).Get(t)

	if err != nil {
		faygo.Debug(err)
		err = SystemFail
		return
	}
	if has {
		err = errors.New("已经存在相同币种名称或者币种符号的虚拟币,请检查是否输入有误")
		return
	}
	n, err := engine.Insert(c_type)
	if err != nil {
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("添加币种失败")
	}
	//重新初始化币种信息
	AllCoinType.Init()
	//构造表信息
	NewCoin(c_type.Id).createTab()
	NewCoinLog(c_type.Id).createTab()
	if c_type.Status == 1 {
		need_reset_issue = true
	}
	return
}

//修改币种信息
func (c *CoinType) Edit(c_type *CoinType) (need_reset_issue bool, err error) {
	engine := xorm.MustDB()
	//查询是否已经存在相同币种符号或者名称的币种
	t := new(CoinType)
	has, err := engine.Where("id=?", c_type.Id).Get(t)

	if err != nil {
		err = SystemFail
		return
	}
	if !has {
		err = errors.New("不存在的币种")
		return
	}
	c_type.Update_time = time.Now().Unix()
	n, err := engine.Where("id=?", c_type.Id).Update(c_type)
	if err != nil {
		err = SystemFail
		return
	}
	if n == 0 {
		err = errors.New("修改失败")
	}
	//重新初始化币种信息
	AllCoinType.Init()
	if t.Status != c_type.Status {
		need_reset_issue = true
	}
	return
}

//获取单个币种的信息
func (c *CoinType) GetCoinTypeById(id int) (c_type *CoinType, err error) {
	engine := xorm.MustDB()
	c_type = new(CoinType)
	has, err := engine.Where("id=?", id).Get(c_type)
	if err != nil {
		err = SystemFail
		return
	}
	if !has {
		err = errors.New("不存在的币种")
	}
	return
}
