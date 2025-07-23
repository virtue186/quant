package market

import (
	"github.com/adshao/go-binance/v2"
	"github.com/markcheno/go-talib"
	"github.com/quant/config"
	"log"
	"strconv"
)

// MarketState 定义市场状态常量
type MarketState int

const (
	StateBull   MarketState = iota // 牛市
	StateBear                      // 熊市
	StateRange                     // 震荡市
	StateUnsure                    // 不确定
)

func (s MarketState) String() string {
	switch s {
	case StateBull:
		return "牛市"
	case StateBear:
		return "熊市"
	case StateRange:
		return "震荡"
	case StateUnsure:
		return "不确定的市场"
	default:
		return "UNKNOWN"
	}
}

func DetermineState(klines []*binance.Kline, conf *config.MarketStateConfig) MarketState {
	// 确定计算所需的最少数据量
	requiredDataLen := conf.LongMAPeriod
	if conf.ADXPeriod > requiredDataLen {
		requiredDataLen = conf.ADXPeriod
	}

	if len(klines) <= requiredDataLen {
		log.Printf("数据长度不足 %d，无法进行市场状态判断。", requiredDataLen)
		return StateUnsure
	}

	highPrices := make([]float64, len(klines))
	lowPrices := make([]float64, len(klines))
	closePrices := make([]float64, len(klines))
	for i, k := range klines {
		var errH, errL, errC error
		highPrices[i], errH = strconv.ParseFloat(k.High, 64)
		lowPrices[i], errL = strconv.ParseFloat(k.Low, 64)
		closePrices[i], errC = strconv.ParseFloat(k.Close, 64)

		if errH != nil || errL != nil || errC != nil {
			log.Printf("数据点解析失败 at index %d: High='%s', Low='%s', Close='%s'. 无法继续分析。", i, k.High, k.Low, k.Close)
			return StateUnsure
		}
	}

	// **【关键修复】** 使用配置文件中正确的 adxPeriod
	adxValues := talib.Adx(highPrices, lowPrices, closePrices, conf.ADXPeriod)
	smaShort := talib.Sma(closePrices, conf.ShortMAPeriod)
	smaLong := talib.Sma(closePrices, conf.LongMAPeriod)

	if adxValues == nil || smaShort == nil || smaLong == nil {
		log.Println("技术指标计算失败 (ADX 或 SMA 返回 nil)。")
		return StateUnsure
	}

	// 获取指标的最后一个有效值
	lastIndex := len(closePrices) - 1
	lastADX := adxValues[len(adxValues)-1]
	lastSmaShort := smaShort[len(smaShort)-1]
	lastSmaLong := smaLong[len(smaLong)-1]

	log.Printf("市场状态判断指标详情 (K线时间: %s):", klines[lastIndex].CloseTime)
	log.Printf("  - 最新收盘价: %s", klines[lastIndex].Close)
	log.Printf("  - ADX(%d) 值: %.2f (阈值: %.2f)", conf.ADXPeriod, lastADX, conf.ADXThreshold)
	log.Printf("  - 短期SMA(%d) 值: %.2f", conf.ShortMAPeriod, lastSmaShort)
	log.Printf("  - 长期SMA(%d) 值: %.2f", conf.LongMAPeriod, lastSmaLong)

	// 核心判断逻辑
	// 1. 使用ADX判断趋势强度
	if lastADX < conf.ADXThreshold {
		log.Println("判断结果: ADX值低于阈值，市场处于震荡状态。")
		return StateRange
	}

	// 2. 若ADX表明存在趋势，则使用均线判断趋势方向
	if lastSmaShort > lastSmaLong {
		log.Println("判断结果: ADX值高于阈值且短期均线上穿长期均线，市场处于牛市趋势。")
		return StateBull
	} else if lastSmaShort < lastSmaLong {
		log.Println("判断结果: ADX值高于阈值且短期均线下穿长期均线，市场处于熊市趋势。")
		return StateBear
	}

	log.Println("判断结果: 指标条件不明确，市场状态不确定。")
	return StateUnsure
}
