package telemetry

import (
	"github.com/ethereum/go-ethereum/common"
	ocrtypes "github.com/DCMMC/libocr/offchainreporting/types"
)

type MonitoringEndpointGenerator interface {
	GenMonitoringEndpoint(addr common.Address) ocrtypes.MonitoringEndpoint
}
