package offchainreporting

import (
	"github.com/DCMMC/chainlink/core/services/job"
	ocrtypes "github.com/DCMMC/libocr/offchainreporting/types"
)

func NewLocalConfig(cfg ValidationConfig, spec job.OffchainReportingOracleSpec) ocrtypes.LocalConfig {
	spec = *job.LoadDynamicConfigVars(cfg, spec)
	lc := ocrtypes.LocalConfig{
		BlockchainTimeout:                      spec.BlockchainTimeout.Duration(),
		ContractConfigConfirmations:            spec.ContractConfigConfirmations,
		SkipContractConfigConfirmations:        cfg.ChainType().IsL2(),
		ContractConfigTrackerPollInterval:      spec.ContractConfigTrackerPollInterval.Duration(),
		ContractConfigTrackerSubscribeInterval: spec.ContractConfigTrackerSubscribeInterval.Duration(),
		ContractTransmitterTransmitTimeout:     cfg.OCRContractTransmitterTransmitTimeout(),
		DatabaseTimeout:                        cfg.OCRDatabaseTimeout(),
		DataSourceTimeout:                      spec.ObservationTimeout.Duration(),
		DataSourceGracePeriod:                  cfg.OCRObservationGracePeriod(),
	}
	if cfg.Dev() {
		// Skips config validation so we can use any config parameters we want.
		// For example to lower contractConfigTrackerPollInterval to speed up tests.
		lc.DevelopmentMode = ocrtypes.EnableDangerousDevelopmentMode
	}
	return lc
}
