package trader

import (
	"fmt"
	b "github.com/quant/binance"
	"github.com/quant/config"
	"github.com/quant/market"
	"github.com/quant/models"
	"github.com/quant/strategy"
	"github.com/quant/utils"
	"github.com/shopspring/decimal"
	"log"
	"time"
)

type Engine struct {
	config        *config.Config
	binanceClient *b.Client
	strategy      *strategy.RollerStrategy
	position      *models.Position
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		config:        cfg,
		binanceClient: b.NewClient(cfg.Binance),
		strategy:      strategy.NewRollerStrategy(&cfg.RollerStrategy),
		position:      &models.Position{IsActive: false}, // 初始无持仓
	}
}

func (e *Engine) Run() {
	log.Println("交易引擎启动...")
	interval, err := time.ParseDuration(e.config.Binance.KlineInterval)
	if err != nil {
		// 提供一个默认值
		log.Printf("K线周期格式错误: %v, 使用默认值15m", err)
		interval = 15 * time.Minute
	}

	// 立即执行一次，然后按周期执行
	e.tick()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		e.tick()
	}
}

func (e *Engine) tick() {
	log.Println("================== 新的交易周期 ==================")

	// 【逻辑缺陷修复】: 通过引用 market 包的公共常量来计算，实现解耦
	requiredKlines := e.config.MarketState.LongMAPeriod + market.ConfirmationBars
	if e.config.MarketState.ADXPeriod > requiredKlines {
		requiredKlines = e.config.MarketState.ADXPeriod
	}
	if e.config.MarketState.MacdSlowPeriod > requiredKlines {
		requiredKlines = e.config.MarketState.MacdSlowPeriod
	}
	requiredKlines += 1
	log.Printf("根据策略需求, 本次请求K线数量: %d", requiredKlines)

	// 获取K线数据
	klines, err := e.binanceClient.FetchKlines(e.config.Binance.TradeSymbol, e.config.Binance.KlineInterval, requiredKlines)
	if err != nil {
		log.Printf("错误: 获取K线数据时发生API错误: %v", err)
		return
	}

	if len(klines) < requiredKlines {
		log.Printf("警告: 从交易所获取的K线数量不足。期望: %d, 实际收到: %d。跳过本周期。", requiredKlines, len(klines))
		return
	}
	log.Printf("成功获取 %d 根K线, 准备进行市场分析...", len(klines))

	// 判断市场状态
	report := market.DetermineState(klines, &e.config.MarketState)
	log.Printf("诊断结果: 市场状态=%s, 原因=%s", report.State, report.Reason)

	// 根据诊断报告进行决策
	if report.State != models.StateBull {
		if e.position.IsActive {
			log.Printf("市场不再是牛市 (当前: %s, 原因: %s)，执行平仓。", report.State, report.Reason)
			e.closePosition(fmt.Sprintf("市场状态改变为 %s", report.State))
		}
		log.Println("策略休眠，等待明确的牛市信号...")
		return
	}

	// 将K线转换为计算格式
	klinesForTalib := utils.ConvertBinanceKlinesToTalib(klines)
	if klinesForTalib == nil {
		log.Println("K线数据转换失败，跳过本周期")
		return
	}

	// 更新动态止损 (如果有持仓)
	if e.position.IsActive {
		e.position.StopLossPrice = e.strategy.UpdateStopLoss(klinesForTalib, e.position.StopLossPrice)
	}

	// 获取策略信号
	signal := e.strategy.CheckSignal(klinesForTalib, e.position)
	log.Printf("获取到策略信号: %s", signal)

	// 执行操作
	switch signal {
	case models.SignalEnterLong:
		e.openPosition(klinesForTalib)
	case models.SignalAddToLong:
		e.addToPosition(klinesForTalib)
	case models.SignalExitLong:
		e.closePosition("策略离场信号")
	}
}

// openPosition 执行开仓逻辑 (模拟)
func (e *Engine) openPosition(klines *models.KLineForTalib) {
	log.Println("[执行操作] 开仓...")
	// 在此实现真实的API下单逻辑
	currentPrice := decimal.NewFromFloat(klines.Close[len(klines.Close)-1])
	quantity := decimal.NewFromFloat(1.0) // 模拟购买1个单位

	e.position.IsActive = true
	e.position.Side = "LONG"
	e.position.Quantity = quantity
	e.position.AvgEntryPrice = currentPrice
	e.position.LastRollPrice = currentPrice
	e.position.StopLossPrice = e.strategy.CalculateInitialStopLoss(klines, currentPrice)

	log.Printf("模拟开仓成功: 价格=%.4f, 数量=%.4f, 初始止损=%.4f",
		e.position.AvgEntryPrice, e.position.Quantity, e.position.StopLossPrice)
}

// addToPosition 执行加仓逻辑 (模拟)
func (e *Engine) addToPosition(klines *models.KLineForTalib) {
	log.Println("[执行操作] 加仓...")
	// 模拟加仓，仓位减半
	newQuantity := e.position.Quantity.Div(decimal.NewFromInt(2))
	currentPrice := decimal.NewFromFloat(klines.Close[len(klines.Close)-1])

	// 更新平均成本
	oldCost := e.position.AvgEntryPrice.Mul(e.position.Quantity)
	newCost := currentPrice.Mul(newQuantity)
	totalQuantity := e.position.Quantity.Add(newQuantity)
	e.position.AvgEntryPrice = oldCost.Add(newCost).Div(totalQuantity)
	e.position.Quantity = totalQuantity
	e.position.LastRollPrice = currentPrice

	log.Printf("模拟加仓成功: 新平均成本=%.4f, 新总数量=%.4f",
		e.position.AvgEntryPrice, e.position.Quantity)
}

// closePosition 执行平仓逻辑 (模拟)
func (e *Engine) closePosition(reason string) {
	log.Printf("[执行操作] 平仓... 原因: %s", reason)
	// 在此实现真实的API平仓逻辑
	e.position.IsActive = false
	e.position.Quantity = decimal.Zero
	e.position.AvgEntryPrice = decimal.Zero
	log.Println("模拟平仓成功，仓位已清空。")
}
