package web_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/DCMMC/chainlink/core/auth"
	"github.com/DCMMC/chainlink/core/bridges"
	"github.com/DCMMC/chainlink/core/internal/cltest"
	"github.com/DCMMC/chainlink/core/web"

	"github.com/stretchr/testify/require"
)

func TestPingController_Show_APICredentials(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start())

	client := app.NewHTTPClient()

	resp, cleanup := client.Get("/v2/ping")
	defer cleanup()
	cltest.AssertServerResponse(t, resp, http.StatusOK)
	body := string(cltest.ParseResponseBody(t, resp))
	require.Equal(t, `{"message":"pong"}`, strings.TrimSpace(body))
}

func TestPingController_Show_ExternalInitiatorCredentials(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start())

	eia := &auth.Token{
		AccessKey: "abracadabra",
		Secret:    "opensesame",
	}
	eir_url := cltest.WebURL(t, "http://localhost:8888")
	eir := &bridges.ExternalInitiatorRequest{
		Name: "bitcoin",
		URL:  &eir_url,
	}

	ei, err := bridges.NewExternalInitiator(eia, eir)
	require.NoError(t, err)
	err = app.BridgeORM().CreateExternalInitiator(ei)
	require.NoError(t, err)

	url := app.Config.ClientNodeURL() + "/v2/ping"
	request, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	request.Header.Set("Content-Type", web.MediaType)
	request.Header.Set("X-Chainlink-EA-AccessKey", eia.AccessKey)
	request.Header.Set("X-Chainlink-EA-Secret", eia.Secret)

	client := http.Client{}
	resp, err := client.Do(request)
	require.NoError(t, err)
	defer resp.Body.Close()

	cltest.AssertServerResponse(t, resp, http.StatusOK)
	body := string(cltest.ParseResponseBody(t, resp))
	require.Equal(t, `{"message":"pong"}`, strings.TrimSpace(body))
}

func TestPingController_Show_NoCredentials(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start())

	client := http.Client{}
	url := app.Config.ClientNodeURL() + "/v2/ping"
	resp, err := client.Get(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
