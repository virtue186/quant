package main

import (
	"github.com/quant/binance"
	"github.com/quant/config"
	"log"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Printf("Error loading config: %v", err)
	}
	// 初始化币安客户端
	binanceClient := binance.NewClient(cfg.Binance)
	log.Println("币安客户端初始化成功。")
	binanceClient.NewKlinesService()

}
