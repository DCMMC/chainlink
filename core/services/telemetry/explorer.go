package telemetry

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/DCMMC/chainlink/core/services/synchronization"
	ocrtypes "github.com/DCMMC/libocr/offchainreporting/types"
)

type ExplorerAgent struct {
	explorerClient synchronization.ExplorerClient
}

// NewExplorerAgent returns a Agent which is just a thin wrapper over
// the explorerClient for now
func NewExplorerAgent(explorerClient synchronization.ExplorerClient) *ExplorerAgent {
	return &ExplorerAgent{explorerClient}
}

// SendLog sends a telemetry log to the explorer
func (t *ExplorerAgent) SendLog(log []byte) {
	t.explorerClient.Send(context.Background(), log, synchronization.ExplorerBinaryMessage)
}

// GenMonitoringEndpoint creates a monitoring endpoint for telemetry
func (t *ExplorerAgent) GenMonitoringEndpoint(addr common.Address) ocrtypes.MonitoringEndpoint {
	return t
}
