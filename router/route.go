package router

import (
	"github.com/henrylee2cn/faygo"
	"wallet/handler"
)

// Route register router in a tree style.
func Route(frame *faygo.Framework) {
	frame.Route(
		frame.NewNamedAPI("展示收益", "POST", "/profit", &handler.Wallet{}),
		frame.NewNamedAPI("获取账单", "POST", "/book", &handler.Book{}),
		frame.NewNamedAPI("领取收益", "POST", "/receive", &handler.Receive_profit{}),
		frame.NewNamedAPI("转账", "POST", "/transfer", &handler.Transfer{}),
		frame.NewNamedAPI("获取手续费", "POST", "/getfee", &handler.GetPoundage{}),
		frame.NewNamedAPI("意见反馈", "POST", "/feedback", &handler.Feedback{}),
		frame.NewNamedAPI("获取算力提升记录", "POST", "/cal_record", &handler.PowerRecord{}),
		frame.NewNamedAPI("获取各个币种的价格", "POST", "/get_price", &handler.GetPrice{}),
		frame.NewGroup("admin",
			frame.NewNamedAPI("获取币种列表", "POST", "/cointypelist", &handler.CoinTypeList{}),
			frame.NewNamedAPI("获取钱包账户列表", "POST", "/account_wallet", &handler.AccountWallet{}),
			frame.NewNamedAPI("用户任务完成回调增加算力记录", "POST", "/add_cal_info", &handler.AddCalculationInfo{}),
			frame.NewNamedAPI("操作币种信息", "POST", "/coin_action", &handler.CoinAction{}),
			frame.NewNamedAPI("添加编辑币种", "POST", "/coin_save", &handler.CoinSave{}),
			frame.NewNamedAPI("通过id获取单个币种", "POST", "/get_cointype", &handler.GetCoinTypeById{}),
		),
	)
}
