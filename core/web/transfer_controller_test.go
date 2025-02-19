package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/DCMMC/chainlink/core/assets"
	"github.com/DCMMC/chainlink/core/internal/cltest"
	"github.com/DCMMC/chainlink/core/services/bulletprooftxmanager"
	"github.com/DCMMC/chainlink/core/store/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransfersController_CreateSuccess_From(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationWithKey(t)
	require.NoError(t, app.Start())

	client := app.NewHTTPClient()
	_, from := cltest.MustInsertRandomKey(t, app.KeyStore.Eth(), 0)

	request := models.SendEtherRequest{
		DestinationAddress: common.HexToAddress("0xFA01FA015C8A5332987319823728982379128371"),
		FromAddress:        from,
		Amount:             *assets.NewEth(100),
	}

	body, err := json.Marshal(&request)
	assert.NoError(t, err)

	resp, cleanup := client.Post("/v2/transfers", bytes.NewBuffer(body))
	t.Cleanup(cleanup)

	errors := cltest.ParseJSONAPIErrors(t, resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, errors.Errors, 0)

	var count int64
	err = app.GetDB().Model(bulletprooftxmanager.EthTx{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestTransfersController_TransferError(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationWithKey(t)
	require.NoError(t, app.Start())

	client := app.NewHTTPClient()
	request := models.SendEtherRequest{
		DestinationAddress: common.HexToAddress("0xFA01FA015C8A5332987319823728982379128371"),
		FromAddress:        common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Amount:             *assets.NewEth(100),
	}

	body, err := json.Marshal(&request)
	assert.NoError(t, err)

	resp, cleanup := client.Post("/v2/transfers", bytes.NewBuffer(body))
	t.Cleanup(cleanup)

	cltest.AssertServerResponse(t, resp, http.StatusBadRequest)
}

func TestTransfersController_JSONBindingError(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationWithKey(t)
	require.NoError(t, app.Start())

	client := app.NewHTTPClient()

	resp, cleanup := client.Post("/v2/transfers", bytes.NewBuffer([]byte(`{"address":""}`)))
	t.Cleanup(cleanup)

	cltest.AssertServerResponse(t, resp, http.StatusBadRequest)
}
