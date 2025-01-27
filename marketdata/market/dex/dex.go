package dex

import (
	"github.com/trustwallet/blockatlas/coin"
	"github.com/trustwallet/blockatlas/marketdata/market"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"github.com/trustwallet/blockatlas/pkg/errors"
	"net/url"
	"strconv"
	"time"
)

var (
	id       = "dex"
	BNBAsset = coin.Binance().Symbol
)

type Market struct {
	market.Market
}

func InitMarket(api string, updateTime string) market.Provider {
	m := &Market{
		Market: market.Market{
			Id:         id,
			Request:    blockatlas.InitClient(api),
			UpdateTime: updateTime,
		},
	}
	return m
}

func (m *Market) GetData() (blockatlas.Tickers, error) {
	var prices []*CoinPrice
	err := m.Get(&prices, "v1/ticker/24hr", url.Values{"limit": {"1000"}})
	if err != nil {
		return nil, err
	}
	rate, err := m.Storage.GetRate(BNBAsset)
	if err != nil {
		return nil, errors.E(err, "rate not found", errors.Params{"asset": BNBAsset})
	}
	result := normalizeTickers(prices, m.GetId())
	result.ApplyRate(1/rate.Rate, blockatlas.DefaultCurrency)
	return result, nil
}

func normalizeTicker(price *CoinPrice, provider string) (*blockatlas.Ticker, error) {
	if price.QuoteAssetName != BNBAsset && price.BaseAssetName != BNBAsset {
		return nil, errors.E("invalid quote/base asset",
			errors.Params{"Symbol": price.BaseAssetName, "QuoteAsset": price.QuoteAssetName})
	}
	value, err := strconv.ParseFloat(price.LastPrice, 64)
	if err != nil {
		return nil, errors.E(err, "normalizeTicker parse value error",
			errors.Params{"LastPrice": price.LastPrice, "Symbol": price.BaseAssetName})
	}
	value24h, err := strconv.ParseFloat(price.PriceChangePercent, 64)
	if err != nil {
		return nil, errors.E(err, "normalizeTicker parse value24h error",
			errors.Params{"PriceChange": price.PriceChangePercent, "Symbol": price.BaseAssetName})
	}
	tokenId := price.BaseAssetName
	if tokenId == BNBAsset {
		tokenId = price.QuoteAssetName
		value = 1.0 / value
	}
	return &blockatlas.Ticker{
		CoinName: BNBAsset,
		CoinType: blockatlas.TypeToken,
		TokenId:  tokenId,
		Price: blockatlas.TickerPrice{
			Value:     value,
			Change24h: value24h,
			Currency:  "BNB",
			Provider:  provider,
		},
		LastUpdate: time.Now(),
	}, nil
}

func normalizeTickers(prices []*CoinPrice, provider string) (tickers blockatlas.Tickers) {
	for _, price := range prices {
		t, err := normalizeTicker(price, provider)
		if err != nil {
			continue
		}
		tickers = append(tickers, t)
	}
	return
}
