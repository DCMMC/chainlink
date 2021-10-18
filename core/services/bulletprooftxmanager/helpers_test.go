package bulletprooftxmanager

import (
	"github.com/DCMMC/chainlink/core/services/eth"
)

func SetEthClientOnEthConfirmer(ethClient eth.Client, ethConfirmer *EthConfirmer) {
	ethConfirmer.ethClient = ethClient
}
