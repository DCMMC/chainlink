package fluxmonitorv2

import (
	"github.com/DCMMC/chainlink/core/internal/gethwrappers/generated/flux_aggregator_wrapper"
	"github.com/DCMMC/chainlink/core/services/log"
	"github.com/DCMMC/chainlink/core/utils"
)

func (fm *FluxMonitor) ExportedPollIfEligible(threshold, absoluteThreshold float64) {
	fm.pollIfEligible(PollRequestTypePoll, NewDeviationChecker(threshold, absoluteThreshold), nil)
}

func (fm *FluxMonitor) ExportedProcessLogs() {
	fm.processLogs()
}

func (fm *FluxMonitor) ExportedBacklog() *utils.BoundedPriorityQueue {
	return fm.backlog
}

func (fm *FluxMonitor) ExportedRoundState() {
	fm.roundState(0)
}

func (fm *FluxMonitor) ExportedRespondToNewRoundLog(log *flux_aggregator_wrapper.FluxAggregatorNewRound, broadcast log.Broadcast) {
	fm.respondToNewRoundLog(*log, broadcast)
}

func (fm *FluxMonitor) ExportedRespondToFlagsRaisedLog() {
	fm.respondToFlagsRaisedLog()
	fm.rotateSelectLoop()
}

func (fm *FluxMonitor) rotateSelectLoop() {
	// the PollRequest is sent to 'rotate' the main select loop, so that new timers will be evaluated
	fm.pollManager.chPoll <- PollRequest{Type: PollRequestTypeUnknown}
}
