package binance

import (
	"context"
	"github.com/adshao/go-binance/v2"
	"github.com/quant/config"
	"log"
)

type Client struct {
	*binance.Client
}

func NewClient(cfg config.BinanceConfig) *Client {
	client := binance.NewClient(cfg.ApiKey, cfg.SecretKey)
	return &Client{client}
}

// FetchKlines 获取指定交易对的K线数据
func (c *Client) FetchKlines(symbol, interval string, limit int) ([]*binance.Kline, error) {
	klines, err := c.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(limit).
		Do(context.Background())

	if err != nil {
		log.Printf("获取K线数据失败: %v", err)
		return nil, err
	}

	return klines, nil
}
