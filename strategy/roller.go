package strategy

import (
	"github.com/markcheno/go-talib"
	"github.com/quant/config"
	"github.com/quant/models"
	"github.com/shopspring/decimal"
	"log"
)

type RollerStrategy struct {
	cfg *config.RollerStrategyConfig
}

func NewRollerStrategy(cfg *config.RollerStrategyConfig) *RollerStrategy {
	return &RollerStrategy{cfg: cfg}
}

// CheckSignal 检查并返回交易信号
func (s *RollerStrategy) CheckSignal(klines *models.KLineForTalib, pos *models.Position) models.Signal {
	if !pos.IsActive {
		return s.checkInitialEntrySignal(klines)
	}
	return s.checkManagementSignal(klines, pos)
}

// checkInitialEntrySignal 检查初始入场信号
func (s *RollerStrategy) checkInitialEntrySignal(klines *models.KLineForTalib) models.Signal {
	// 2. 小周期入场触发
	emaShort := talib.Ema(klines.Close, s.cfg.EntryEmaShortPeriod)
	emaLong := talib.Ema(klines.Close, s.cfg.EntryEmaLongPeriod)

	// 数据不足
	if len(emaShort) < 2 || len(emaLong) < 2 {
		return models.SignalDoNothing
	}

	// 检查金叉
	prevEmaShort := emaShort[len(emaShort)-2]
	prevEmaLong := emaLong[len(emaLong)-2]
	currEmaShort := emaShort[len(emaShort)-1]
	currEmaLong := emaLong[len(emaLong)-1]

	isGoldenCross := prevEmaShort <= prevEmaLong && currEmaShort > currEmaLong

	if isGoldenCross {
		// 3. 动能和成交量确认
		smaVolume := talib.Sma(klines.Volume, s.cfg.VolumeSmaPeriod)
		if len(smaVolume) < 1 {
			return models.SignalDoNothing
		}
		lastVolume := klines.Volume[len(klines.Volume)-1]
		avgVolume := smaVolume[len(smaVolume)-1]
		requiredVolume := avgVolume * s.cfg.VolumeMultiplier

		if lastVolume > requiredVolume {
			log.Printf("[策略信号] 发现金叉且成交量放大. LastVol: %.2f, AvgVol: %.2f, RequiredVol: %.2f", lastVolume, avgVolume, requiredVolume)
			return models.SignalEnterLong
		}
	}

	return models.SignalDoNothing
}

// checkManagementSignal 检查持仓管理信号 (滚仓或离场)
func (s *RollerStrategy) checkManagementSignal(klines *models.KLineForTalib, pos *models.Position) models.Signal {
	currentPriceFloat := klines.Close[len(klines.Close)-1]
	currentPrice := decimal.NewFromFloat(currentPriceFloat)

	// 1. 检查是否触发止损
	if currentPrice.LessThanOrEqual(pos.StopLossPrice) {
		log.Printf("[策略信号] 价格 %.4f 触及止损位 %.4f, 准备离场。", currentPrice, pos.StopLossPrice)
		return models.SignalExitLong
	}

	// 2. 检查是否触发止盈离场 (这里用一个简单的均线跌破作为例子)
	emaExit := talib.Ema(klines.Close, s.cfg.EntryEmaLongPeriod)
	if len(emaExit) > 0 {
		exitPrice := decimal.NewFromFloat(emaExit[len(emaExit)-1])
		if currentPrice.LessThan(exitPrice) {
			log.Printf("[策略信号] 价格 %.4f 跌破长期均线 %.4f, 准备止盈离场。", currentPrice, exitPrice)
			return models.SignalExitLong
		}
	}

	// 3. 检查滚仓信号
	if pos.Quantity.IsZero() {
		return models.SignalDoNothing
	}
	unrealizedProfit := currentPrice.Sub(pos.AvgEntryPrice).Mul(pos.Quantity)
	initialCost := pos.AvgEntryPrice.Mul(pos.Quantity)
	profitRatio := unrealizedProfit.Div(initialCost)

	// 检查盈利是否超过阈值
	if profitRatio.GreaterThan(decimal.NewFromFloat(s.cfg.RollProfitThreshold)) {
		// 检查价格是否比上次加仓点有显著上涨，防止在同一水平反复触发
		requiredPrice := pos.LastRollPrice.Mul(decimal.NewFromFloat(1 + s.cfg.RollCheckPriceGap))
		if currentPrice.GreaterThan(requiredPrice) {
			log.Printf("[策略信号] 盈利 %.2f%% > 阈值 %.2f%%, 且当前价格 %.4f > 上次滚动点 %.4f * (1+%.2f). 准备加仓。",
				profitRatio.Mul(decimal.NewFromInt(100)), s.cfg.RollProfitThreshold*100, currentPrice, pos.LastRollPrice, s.cfg.RollCheckPriceGap)
			return models.SignalAddToLong
		}
	}

	return models.SignalDoNothing
}

// CalculateInitialStopLoss 计算初始止损价
func (s *RollerStrategy) CalculateInitialStopLoss(klines *models.KLineForTalib, entryPrice decimal.Decimal) decimal.Decimal {
	atr := talib.Atr(klines.High, klines.Low, klines.Close, s.cfg.AtrPeriod)
	if len(atr) == 0 {
		// 如果ATR计算失败，使用一个固定的百分比作为备用方案
		return entryPrice.Mul(decimal.NewFromFloat(0.98)) // 2% 止损
	}
	lastAtr := decimal.NewFromFloat(atr[len(atr)-1])
	stopLoss := entryPrice.Sub(lastAtr.Mul(decimal.NewFromFloat(s.cfg.AtrMultiplier)))
	return stopLoss
}

// UpdateStopLoss 更新动态止损 (吊灯止损)
func (s *RollerStrategy) UpdateStopLoss(klines *models.KLineForTalib, currentStopLoss decimal.Decimal) decimal.Decimal {
	atr := talib.Atr(klines.High, klines.Low, klines.Close, s.cfg.AtrPeriod)
	if len(atr) == 0 {
		return currentStopLoss
	}
	lastAtr := decimal.NewFromFloat(atr[len(atr)-1])
	currentPrice := decimal.NewFromFloat(klines.Close[len(klines.Close)-1])

	potentialNewStop := currentPrice.Sub(lastAtr.Mul(decimal.NewFromFloat(s.cfg.AtrMultiplier)))

	// 止损只上移，不下移
	if potentialNewStop.GreaterThan(currentStopLoss) {
		log.Printf("[止损更新] 新的动态止损位: %.4f (原: %.4f)", potentialNewStop, currentStopLoss)
		return potentialNewStop
	}
	return currentStopLoss
}
