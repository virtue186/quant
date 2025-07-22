package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	MarketState MarketStateConfig `mapstructure:"marketState"`
	Binance     BinanceConfig     `mapstructure:"binance"` // 新增
}

type BinanceConfig struct {
	ApiKey        string `mapstructure:"apiKey"`
	SecretKey     string `mapstructure:"secretKey"`
	TradeSymbol   string `mapstructure:"tradeSymbol"`
	KlineInterval string `mapstructure:"klineInterval"`
}
type MarketStateConfig struct {
	ShortMAPeriod int     // 短期移动平均线的计算周期
	LongMAPeriod  int     // 长期移动平均线的计算周期
	ADXPeriod     int     // ADX指标的计算周期
	ADXThreshold  float64 // 判断市场是否进入震荡市的ADX阈值
}

var Cfg *Config

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(path)
	// 支持环境变量替换配置文件参数
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	Cfg = &cfg
	return &cfg, nil
}
