package marketdata

import (
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/trustwallet/blockatlas/marketdata/market"
	"github.com/trustwallet/blockatlas/marketdata/market/coingecko"
	cmc "github.com/trustwallet/blockatlas/marketdata/market/coinmarketcap"
	"github.com/trustwallet/blockatlas/marketdata/market/compound"
	"github.com/trustwallet/blockatlas/marketdata/market/dex"
	"github.com/trustwallet/blockatlas/pkg/errors"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"github.com/trustwallet/blockatlas/storage"
)

var marketProviders market.Providers

func InitMarkets(storage storage.Market) {
	marketProviders = market.Providers{
		// Add Market Quote Providers:
		0: dex.InitMarket(
			viper.GetString("market.dex.api"),
			viper.GetString("market.dex.quote_update_time"),
		),
		1: cmc.InitMarket(
			viper.GetString("market.cmc.api"),
			viper.GetString("market.cmc.api_key"),
			viper.GetString("market.quote_update_time"),
		),
		2: compound.InitMarket(
			viper.GetString("market.compound.api"),
			viper.GetString("market.quote_update_time"),
		),
		3: coingecko.InitMarket(
			viper.GetString("market.coingecko.api"),
			viper.GetString("market.quote_update_time"),
		),
	}
	addMarkets(storage, marketProviders)
}

func addMarkets(storage storage.Market, ps market.Providers) {
	c := cron.New()
	for _, p := range ps {
		scheduleTasks(storage, p, c)
	}
	c.Start()
}

func runMarket(storage storage.Market, p market.Provider) error {
	data, err := p.GetData()
	if err != nil {
		return errors.E(err, "GetData")
	}
	for _, result := range data {
		err = storage.SaveTicker(result, marketProviders)
		if err != nil {
			logger.Error(errors.E(err, "SaveTicker",
				errors.Params{"result": result}))
		}
	}
	logger.Info("Market data result", logger.Params{"markets": len(data), "provider": p.GetId()})
	return nil
}
