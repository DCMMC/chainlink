package synchronization_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/onsi/gomega"
	"github.com/DCMMC/chainlink/core/internal/cltest"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/csakey"
	ksmocks "github.com/DCMMC/chainlink/core/services/keystore/mocks"
	"github.com/DCMMC/chainlink/core/services/synchronization"
	"github.com/DCMMC/chainlink/core/services/synchronization/mocks"
	telemPb "github.com/DCMMC/chainlink/core/services/synchronization/telem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tevino/abool"
)

func TestTelemetryIngressClient_Send_HappyPath(t *testing.T) {

	// Create mocks
	telemClient := new(mocks.TelemClient)
	csaKeystore := new(ksmocks.CSA)

	// Set mock handlers for keystore
	key := cltest.DefaultCSAKey
	keyList := []csakey.KeyV2{key}
	csaKeystore.On("GetAll").Return(keyList, nil)

	// Wire up the telem ingress client
	url := &url.URL{}
	serverPubKeyHex := "33333333333"
	telemIngressClient := synchronization.NewTestTelemetryIngressClient(url, serverPubKeyHex, csaKeystore, false, telemClient)
	require.NoError(t, telemIngressClient.Start())
	defer telemIngressClient.Close()

	// Create the telemetry payload
	telemetry := []byte("101010")
	address := common.HexToAddress("0xa")
	telemPayload := synchronization.TelemPayload{
		Ctx:             context.Background(),
		Telemetry:       telemetry,
		ContractAddress: address,
	}

	// Assert the telemetry payload is correctly sent to wsrpc
	called := abool.New()
	telemClient.On("Telem", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
		called.Set()
		telemReq := args.Get(1).(*telemPb.TelemRequest)
		assert.Equal(t, telemPayload.ContractAddress.String(), telemReq.Address)
		assert.Equal(t, telemPayload.Telemetry, telemReq.Telemetry)
	})

	// Send telemetry
	telemIngressClient.Send(telemPayload)

	// Wait for the telemetry to be handled
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		return called.IsSet()
	}).Should(gomega.BeTrue())
}
