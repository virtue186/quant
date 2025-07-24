package main

import (
	"github.com/quant/config"
	"github.com/quant/trader"
	"log"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("./config/")
	if err != nil {
		log.Fatalf("错误：无法加载配置: %v", err)
	}
	log.Println("配置加载成功。")

	// 2. 初始化并运行交易引擎
	engine := trader.NewEngine(cfg)
	engine.Run()

}
