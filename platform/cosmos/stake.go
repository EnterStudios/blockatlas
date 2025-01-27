package cosmos

import (
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"github.com/trustwallet/blockatlas/pkg/errors"
	"github.com/trustwallet/blockatlas/pkg/logger"
	services "github.com/trustwallet/blockatlas/services/assets"
	"strconv"
	"time"
)

func (p *Platform) GetValidators() (blockatlas.ValidatorPage, error) {
	results := make(blockatlas.ValidatorPage, 0)
	validators, err := p.client.GetValidators()
	if err != nil {
		return nil, err
	}
	pool, err := p.client.GetPool()
	if err != nil {
		return nil, err
	}

	inflation, err := p.client.GetInflation()
	if err != nil {
		return nil, err
	}
	inflationValue, err := strconv.ParseFloat(inflation.Result, 32)
	if err != nil {
		return nil, errors.E("error to parse inflationValue to float", errors.TypePlatformUnmarshal).PushToSentry()
	}

	for _, validator := range validators.Result {
		results = append(results, normalizeValidator(validator, pool.Pool, inflationValue))
	}

	return results, nil
}

func (p *Platform) GetDetails() blockatlas.StakingDetails {
	//TODO: Find a way to have a dynamic
	return blockatlas.StakingDetails{
		Reward:        blockatlas.StakingReward{Annual: 11},
		MinimumAmount: blockatlas.Amount("0"),
		LockTime:      1814400,
		Type:          blockatlas.DelegationTypeDelegate,
	}
}

func (p *Platform) GetDelegations(address string) (blockatlas.DelegationsPage, error) {
	results := make(blockatlas.DelegationsPage, 0)
	delegations, err := p.client.GetDelegations(address)
	if err != nil {
		return nil, err
	}
	unbondingDelegations, err := p.client.GetUnbondingDelegations(address)
	if err != nil {
		return nil, err
	}
	if delegations.List == nil && unbondingDelegations.List == nil {
		return results, nil
	}
	validators, err := services.GetValidatorsMap(p)
	if err != nil {
		return nil, err
	}
	results = append(results, NormalizeDelegations(delegations.List, validators)...)
	results = append(results, NormalizeUnbondingDelegations(unbondingDelegations.List, validators)...)

	return results, nil
}

func (p *Platform) UndelegatedBalance(address string) (string, error) {
	account, err := p.client.GetAccount(address)
	if err != nil {
		return "0", err
	}
	for _, coin := range account.Account.Value.Coins {
		if coin.Denom == UndelegateDenom {
			return coin.Amount, nil
		}
	}
	return "0", nil
}

func NormalizeDelegations(delegations []Delegation, validators blockatlas.ValidatorMap) []blockatlas.Delegation {
	results := make([]blockatlas.Delegation, 0)
	for _, v := range delegations {
		validator, ok := validators[v.ValidatorAddress]
		if !ok {
			logger.Error(errors.E("Validator not found", errors.Params{"address": v.ValidatorAddress, "platform": "cosmos", "delegation": v.DelegatorAddress}))
			continue
		}
		delegation := blockatlas.Delegation{
			Delegator: validator,
			Value:     v.Value(),
			Status:    blockatlas.DelegationStatusActive,
		}
		results = append(results, delegation)
	}
	return results
}

func NormalizeUnbondingDelegations(delegations []UnbondingDelegation, validators blockatlas.ValidatorMap) []blockatlas.Delegation {
	results := make([]blockatlas.Delegation, 0)
	for _, v := range delegations {
		for _, entry := range v.Entries {
			validator, ok := validators[v.ValidatorAddress]
			if !ok {
				logger.Error(errors.E("Validator not found", errors.Params{"address": v.ValidatorAddress, "platform": "cosmos", "delegation": v.DelegatorAddress}))
				continue
			}
			t, _ := time.Parse(time.RFC3339, entry.CompletionTime)
			delegation := blockatlas.Delegation{
				Delegator: validator,
				Value:     entry.Balance,
				Status:    blockatlas.DelegationStatusPending,
				Metadata: blockatlas.DelegationMetaDataPending{
					AvailableDate: uint(t.Unix()),
				},
			}
			results = append(results, delegation)
		}
	}
	return results
}

func normalizeValidator(v Validator, p Pool, inflation float64) (validator blockatlas.Validator) {
	reward := CalculateAnnualReward(p, inflation, v)
	return blockatlas.Validator{
		Status: v.Status == 2,
		ID:     v.Address,
		Details: blockatlas.StakingDetails{
			Reward:        blockatlas.StakingReward{Annual: reward},
			MinimumAmount: "0",
			LockTime:      1814400,
			Type:          blockatlas.DelegationTypeDelegate,
		},
	}
}

func CalculateAnnualReward(p Pool, inflation float64, validator Validator) float64 {
	notBondedTokens, err := strconv.ParseFloat(p.NotBondedTokens, 32)
	if err != nil {
		return 0
	}

	bondedTokens, err := strconv.ParseFloat(p.BondedTokens, 32)
	if err != nil {
		return 0
	}

	commission, err := strconv.ParseFloat(validator.Commission.Commision.Rate, 32)
	if err != nil {
		return 0
	}
	result := (notBondedTokens + bondedTokens) / bondedTokens * inflation
	return (result - (result * commission)) * 100
}
