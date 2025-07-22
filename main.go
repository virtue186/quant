package main

import (
	"fmt"
	"github.com/markcheno/go-talib"
	"log"
)

func main() {
	// 示例的收盘价数据
	closePrices := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}

	// 计算 5 日简单移动平均线 (SMA)
	sma := talib.Sma(closePrices, 5)
	if sma == nil {
		log.Fatal("SMA calculation failed")
	}

	fmt.Println("5-day SMA:", sma)
}
