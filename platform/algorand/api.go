package algorand

import (
	"github.com/spf13/viper"

	"strconv"

	"github.com/trustwallet/blockatlas/coin"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
)

type Platform struct {
	client Client
}

func (p *Platform) Init() error {
	p.client = Client{blockatlas.InitClient(viper.GetString("algorand.api"))}
	p.client.Headers["x-api-key"] = viper.GetString("algorand.key")
	return nil
}

func (p *Platform) Coin() coin.Coin {
	return coin.Coins[coin.ALGO]
}

func (p *Platform) CurrentBlockNumber() (int64, error) {
	return p.client.GetLatestBlock()
}

func (p *Platform) GetBlockByNumber(num int64) (*blockatlas.Block, error) {
	txs, err := p.client.GetTxsInBlock(num)
	if err != nil {
		return nil, err
	}

	return &blockatlas.Block{
		Number: num,
		Txs:    NormalizeTxs(txs),
	}, nil
}

func (p *Platform) GetTxsByAddress(address string) (blockatlas.TxPage, error) {
	txs, err := p.client.GetTxsOfAddress(address)
	if err != nil {
		return nil, err
	}
	return NormalizeTxs(txs), nil
}

func (p *Platform) GetDetails() blockatlas.StakingDetails {
	//TODO: Find a way to have a dynamic
	return blockatlas.StakingDetails{
		Reward:        blockatlas.StakingReward{Annual: 7},
		MinimumAmount: blockatlas.Amount("0"),
		LockTime:      0,
	}
}

func (p *Platform) UndelegatedBalance(address string) (string, error) {
	acc, err := p.client.GetAccount(address)
	if err != nil {
		return "0", err
	}
	return strconv.FormatUint(acc.Amount, 10), nil
}

func (p *Platform) GetValidators() (blockatlas.ValidatorPage, error) {
	return make(blockatlas.ValidatorPage, 0), nil
}

func (p *Platform) GetDelegations(address string) (blockatlas.DelegationsPage, error) {
	return make(blockatlas.DelegationsPage, 0), nil
}

func NormalizeTxs(txs []Transaction) []blockatlas.Tx {
	result := make([]blockatlas.Tx, 0)

	for _, tx := range txs {
		if normalized, ok := Normalize(tx); ok {
			result = append(result, normalized)
		}
	}

	return result
}

func Normalize(tx Transaction) (result blockatlas.Tx, ok bool) {

	if tx.Type != TransactionTypePay {
		return result, false
	}

	return blockatlas.Tx{
		ID:     tx.Hash,
		Coin:   coin.ALGO,
		From:   tx.From,
		To:     tx.Payment.To,
		Fee:    blockatlas.Amount(strconv.Itoa(int(tx.Fee))),
		Date:   int64(tx.Timestamp),
		Block:  tx.Round,
		Status: blockatlas.StatusCompleted,
		Type:   blockatlas.TxTransfer,
		Meta: blockatlas.Transfer{
			Value:    blockatlas.Amount(strconv.Itoa(int(tx.Payment.Amount))),
			Symbol:   coin.Coins[coin.ALGO].Symbol,
			Decimals: coin.Coins[coin.ALGO].Decimals,
		},
	}, true
}
