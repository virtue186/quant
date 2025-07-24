package utils

import (
	"github.com/adshao/go-binance/v2"
	"github.com/quant/models"
	"log"
	"strconv"
)

// ConvertBinanceKlinesToTalib 转换币安K线为talib接受的格式
func ConvertBinanceKlinesToTalib(klines []*binance.Kline) *models.KLineForTalib {
	count := len(klines)
	if count == 0 {
		return nil
	}

	klineTalib := &models.KLineForTalib{
		High:   make([]float64, count),
		Low:    make([]float64, count),
		Close:  make([]float64, count),
		Volume: make([]float64, count),
	}

	for i, k := range klines {
		high, errH := strconv.ParseFloat(k.High, 64)
		low, errL := strconv.ParseFloat(k.Low, 64)
		close, errC := strconv.ParseFloat(k.Close, 64)
		volume, errV := strconv.ParseFloat(k.Volume, 64)

		if errH != nil || errL != nil || errC != nil || errV != nil {
			log.Printf("数据点解析失败 at index %d, 跳过该点", i)
			// 在实际应用中可能需要更复杂的错误处理
			continue
		}
		klineTalib.High[i] = high
		klineTalib.Low[i] = low
		klineTalib.Close[i] = close
		klineTalib.Volume[i] = volume
	}
	return klineTalib
}
