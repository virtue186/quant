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

func DetermineState(klines []*binance.Kline, config *config.Config) MarketState {
	requiredDataLen := config.MarketState.LongMAPeriod
	if config.MarketState.ADXPeriod > requiredDataLen {
		requiredDataLen = config.MarketState.ADXPeriod
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
		// 将 string 转换为 float64
		highPrices[i], errH = strconv.ParseFloat(k.High, 64)
		lowPrices[i], errL = strconv.ParseFloat(k.Low, 64)
		closePrices[i], errC = strconv.ParseFloat(k.Close, 64)

		if errH != nil || errL != nil || errC != nil {
			log.Printf("数据点解析失败 at index %d: High='%s', Low='%s', Close='%s'. 无法继续分析。", i, k.High, k.Low, k.Close)
			return StateUnsure // 遇到无法解析的数据，立即返回不确定状态
		}
	}

	adxValues := talib.Adx(highPrices, lowPrices, closePrices, config.MarketState.ShortMAPeriod)
	smaShort := talib.Sma(closePrices, config.MarketState.ShortMAPeriod)
	smaLong := talib.Sma(closePrices, config.MarketState.LongMAPeriod)
	if adxValues == nil {
		log.Println("ADX 指标计算失败。")
		return StateUnsure
	}
	if smaShort == nil || smaLong == nil {
		log.Println("SMA 指标计算失败。")
		return StateUnsure
	}

	lastIndex := len(closePrices) - 1
	lastSmaShort := smaShort[lastIndex]
	lastSmaLong := smaLong[lastIndex]
	lastADX := adxValues[lastIndex]

	if lastADX < config.MarketState.ADXThreshold {
		return StateRange
	}

	if lastSmaShort > lastSmaLong {
		return StateBull
	} else if lastSmaShort < lastSmaLong {
		return StateBear
	}

	return StateUnsure
}
