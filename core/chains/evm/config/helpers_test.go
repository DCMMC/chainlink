package config

import "github.com/DCMMC/chainlink/core/chains/evm/types"

func PersistedCfgPtr(cfg ChainScopedConfig) *types.ChainCfg {
	return &cfg.(*chainScopedConfig).persistedCfg
}

func ChainSpecificConfigDefaultSets() map[int64]chainSpecificConfigDefaultSet {
	return chainSpecificConfigDefaultSets
}
