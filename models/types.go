package models

import "github.com/shopspring/decimal"

// MarketState 定义了市场的宏观状态
type MarketState int

const (
	StateBull   MarketState = iota // 趋势牛市
	StateBear                      // 趋势熊市
	StateRange                     // 震荡市
	StateUnsure                    // 不确定状态
)

// UncertaintyReason 精确定义了市场不确定的具体原因
type UncertaintyReason int

const (
	ReasonNone                    UncertaintyReason = iota // 无不确定性
	ReasonInsufficientData                                 // 数据长度不足
	ReasonIndicatorError                                   // 技术指标计算失败
	ReasonTrendConfirmationFailed                          // 趋势信号未能得到确认（例如，未站稳）
	ReasonADXWeak                                          // ADX值过低，无明显趋势
	ReasonADXDeclining                                     // ADX正在下降，趋势可能衰竭
	ReasonMACDConflict                                     // MACD指标与趋势方向冲突
)

// MarketStatusReport 是市场状态判断的最终输出，像一份诊断报告
type MarketStatusReport struct {
	State  MarketState
	Reason UncertaintyReason
}

// Signal 定义了策略信号的类型
type Signal string

const (
	SignalDoNothing Signal = "DO_NOTHING"
	SignalEnterLong Signal = "ENTER_LONG"
	SignalAddToLong Signal = "ADD_TO_LONG"
	SignalExitLong  Signal = "EXIT_LONG"
)

// Position 代表一个持仓
type Position struct {
	Symbol        string
	Side          string // "LONG"
	AvgEntryPrice decimal.Decimal
	Quantity      decimal.Decimal
	StopLossPrice decimal.Decimal
	LastRollPrice decimal.Decimal // 记录上一次滚仓时的价格
	IsActive      bool
}

// KLineForTalib 是一个为 talib 计算优化的K线结构体
type KLineForTalib struct {
	High   []float64
	Low    []float64
	Close  []float64
	Volume []float64
}

// Stringer for nice logging
func (s MarketState) String() string {
	return []string{"牛市", "熊市", "震荡", "不确定"}[s]
}

func (r UncertaintyReason) String() string {
	return []string{
		"无", "数据不足", "指标计算错误",
		"趋势未能确认", "ADX趋势过弱", "ADX趋势衰竭",
		"MACD信号冲突",
	}[r]
}
