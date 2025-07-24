package market

import (
	"github.com/adshao/go-binance/v2"
	"github.com/markcheno/go-talib"
	"github.com/quant/config"
	"github.com/quant/models"
	"github.com/quant/utils"
	"log"
	"time"
)

const (
	// ConfirmationBars 定义了趋势确认所需的K线数量
	ConfirmationBars = 2
)

// DetermineState 判断市场状态 (V4 "可解释性"重构版)
func DetermineState(klines []*binance.Kline, conf *config.MarketStateConfig) models.MarketStatusReport {
	log.Println("--- 开始新一轮市场状态分析 ---")

	// 步骤 1: 检查数据是否充足
	requiredDataLen := calculateRequiredDataLength(conf)
	if len(klines) <= requiredDataLen {
		log.Printf("[步骤 1/5] 失败: 数据长度不足。需要: >%d, 实际: %d。", requiredDataLen, len(klines))
		return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonInsufficientData}
	}
	log.Printf("[步骤 1/5] 通过: 数据长度充足 (%d)。", len(klines))

	// 步骤 2: 数据转换和指标计算
	klineData := utils.ConvertBinanceKlinesToTalib(klines)
	emaShort := talib.Ema(klineData.Close, conf.ShortMAPeriod)
	emaLong := talib.Ema(klineData.Close, conf.LongMAPeriod)
	adxValues := talib.Adx(klineData.High, klineData.Low, klineData.Close, conf.ADXPeriod)
	macd, macdSignal, _ := talib.Macd(klineData.Close, conf.MacdFastPeriod, conf.MacdSlowPeriod, conf.MacdSignalPeriod)

	if adxValues == nil || emaShort == nil || emaLong == nil || macd == nil || macdSignal == nil || len(adxValues) < 2 {
		log.Println("[步骤 2/5] 失败: 技术指标计算返回nil或数据不足。")
		return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonIndicatorError}
	}
	log.Println("[步骤 2/5] 通过: 所有技术指标计算成功。")

	// 提取最新指标值
	lastADX := adxValues[len(adxValues)-1]
	lastEmaShort := emaShort[len(emaShort)-1]
	lastEmaLong := emaLong[len(emaLong)-1]
	lastMacd := macd[len(macd)-1]
	lastMacdSignal := macdSignal[len(macdSignal)-1]

	// 打印详细指标，方便复盘
	logRawIndicators(klines[len(klines)-1], conf, lastADX, adxValues, lastEmaShort, lastEmaLong, lastMacd, lastMacdSignal)

	// 步骤 3: 检查趋势强度 (ADX)
	log.Printf("[步骤 3/5] 检查ADX强度... ADX(%.2f) vs 阈值(%.1f)", lastADX, conf.ADXThreshold)
	if lastADX < conf.ADXThreshold {
		log.Println("          └─ 结论: ADX值过低，市场处于【震荡】。分析结束。")
		return models.MarketStatusReport{State: models.StateRange, Reason: models.ReasonADXWeak}
	}
	log.Println("          └─ 结论: 趋势强度足够。继续分析趋势动能...")

	// 步骤 4: 检查趋势动能 (ADX趋势)
	adxIsDeclining := true
	if len(adxValues) > conf.AdxDeclineBars {
		for i := 1; i <= conf.AdxDeclineBars; i++ {
			if adxValues[len(adxValues)-i] > adxValues[len(adxValues)-i-1] {
				adxIsDeclining = false // 只要有一根不是下降，就认为动能未衰竭
				break
			}
		}
	} else {
		adxIsDeclining = false // 数据不足，无法判断
	}

	log.Printf("[步骤 4/5] 检查ADX动能... 是否连续 %d 根下降？ -> %t", conf.AdxDeclineBars, adxIsDeclining)
	if adxIsDeclining {
		log.Println("          └─ 结论: ADX连续下降，趋势动力衰竭，市场【不确定】。分析结束。")
		return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonADXDeclining}
	}
	log.Println("          └─ 结论: 趋势动能强劲。继续分析趋势方向...")

	// 步骤 5: 检查趋势方向与共振 (EMA & MACD)
	log.Println("[步骤 5/5] 检查趋势方向...")
	if lastEmaShort > lastEmaLong {
		log.Println("          ├─ EMA判断: 短期线上穿长期线，初步判断为【多头】。")
		log.Printf("          ├─ MACD确认: MACD Line(%.4f) vs Signal Line(%.4f)", lastMacd, lastMacdSignal)
		if lastMacd < lastMacdSignal {
			log.Println("          │   └─ 结论: MACD死叉或位于信号线之下，与EMA方向冲突，市场【不确定】。分析结束。")
			return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonMACDConflict}
		}
		log.Println("          │   └─ 结论: MACD信号一致。继续检查趋势是否站稳...")

		isConfirmed := checkConfirmation(klineData, emaShort, true)
		log.Printf("          └─ 站稳确认: 过去 %d 根K线是否持续站稳在短期EMA之上？ -> %t", ConfirmationBars, isConfirmed)
		if isConfirmed {
			log.Println("             └─ 最终结论: 市场处于明确的【牛市】趋势。")
			return models.MarketStatusReport{State: models.StateBull, Reason: models.ReasonNone}
		} else {
			log.Println("             └─ 最终结论: 趋势未能站稳，市场【不确定】。")
			return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonTrendConfirmationFailed}
		}
	} else if lastEmaShort < lastEmaLong {
		log.Println("          ├─ EMA判断: 短期线下穿长期线，初步判断为【空头】。")
		log.Printf("          ├─ MACD确认: MACD Line(%.4f) vs Signal Line(%.4f)", lastMacd, lastMacdSignal)
		if lastMacd > lastMacdSignal {
			log.Println("          │   └─ 结论: MACD金叉或位于信号线之上，与EMA方向冲突，市场【不确定】。分析结束。")
			return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonMACDConflict}
		}
		log.Println("          │   └─ 结论: MACD信号一致。继续检查趋势是否站稳...")

		isConfirmed := checkConfirmation(klineData, emaShort, false)
		log.Printf("          └─ 站稳确认: 过去 %d 根K线是否持续站稳在短期EMA之下？ -> %t", ConfirmationBars, isConfirmed)
		if isConfirmed {
			log.Println("             └─ 最终结论: 市场处于明确的【熊市】趋势。")
			return models.MarketStatusReport{State: models.StateBear, Reason: models.ReasonNone}
		} else {
			log.Println("             └─ 最终结论: 趋势未能站稳，市场【不确定】。")
			return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonTrendConfirmationFailed}
		}
	}

	log.Println("步骤 5 结论: EMA均线缠绕，方向不明确，市场【不确定】。")
	return models.MarketStatusReport{State: models.StateUnsure, Reason: models.ReasonTrendConfirmationFailed}
}

// --- 以下为辅助函数 ---

func calculateRequiredDataLength(conf *config.MarketStateConfig) int {
	// 这是一个辅助函数，用于集中计算所需的数据长度
	maxPeriod := conf.LongMAPeriod
	if conf.ADXPeriod > maxPeriod {
		maxPeriod = conf.ADXPeriod
	}
	if conf.MacdSlowPeriod > maxPeriod {
		maxPeriod = conf.MacdSlowPeriod
	}
	return maxPeriod + ConfirmationBars
}

func logRawIndicators(lastKline *binance.Kline, conf *config.MarketStateConfig, lastADX float64, adxValues []float64, lastEmaShort, lastEmaLong, lastMacd, lastMacdSignal float64) {
	klineTime := time.Unix(0, lastKline.CloseTime*int64(time.Millisecond))
	adxTrend := "→"
	if len(adxValues) >= 2 {
		if adxValues[len(adxValues)-1] > adxValues[len(adxValues)-2] {
			adxTrend = "↑"
		} else if adxValues[len(adxValues)-1] < adxValues[len(adxValues)-2] {
			adxTrend = "↓"
		}
	}
	log.Println("--- 原始指标快照 ---")
	log.Printf("  K线时间: %s, 收盘价: %s", klineTime.Format("2006-01-02 15:04:05"), lastKline.Close)
	log.Printf("  ADX(%d): %.2f (阈值 >%.1f) | 趋势: %s", conf.ADXPeriod, lastADX, conf.ADXThreshold, adxTrend)
	log.Printf("  EMA(%d): %.4f | EMA(%d): %.4f", conf.ShortMAPeriod, lastEmaShort, conf.LongMAPeriod, lastEmaLong)
	log.Printf("  MACD(%d,%d,%d): MACD=%.4f, Signal=%.4f", conf.MacdFastPeriod, conf.MacdSlowPeriod, conf.MacdSignalPeriod, lastMacd, lastMacdSignal)
	log.Println("--------------------")
}

func checkConfirmation(klineData *models.KLineForTalib, emaShort []float64, isBull bool) bool {
	// 辅助函数，用于检查趋势是否站稳
	lastIndex := len(klineData.Close) - 1
	for i := 1; i <= ConfirmationBars; i++ {
		isConfirmed := false
		if isBull {
			// 牛市确认：收盘价必须在短期EMA之上
			isConfirmed = klineData.Close[lastIndex-i] > emaShort[len(emaShort)-1-i]
		} else {
			// 熊市确认：收盘价必须在短期EMA之下
			isConfirmed = klineData.Close[lastIndex-i] < emaShort[len(emaShort)-1-i]
		}
		if !isConfirmed {
			return false // 只要有一根K线不满足，就确认失败
		}
	}
	return true
}

func formatTrend(isRising bool) string {
	if isRising {
		return "上升"
	}
	return "下降"
}
