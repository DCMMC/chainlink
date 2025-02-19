package presenters_test

import (
	"fmt"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/lib/pq"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"

	"github.com/DCMMC/chainlink/core/assets"
	"github.com/DCMMC/chainlink/core/services/job"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/ethkey"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/DCMMC/chainlink/core/services/pipeline"
	"github.com/DCMMC/chainlink/core/store/models"
	"github.com/DCMMC/chainlink/core/web/presenters"
)

func TestJob(t *testing.T) {
	// Used in most tests
	timestamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	contractAddress, err := ethkey.NewEIP55Address("0x9E40733cC9df84636505f4e6Db28DCa0dC5D1bba")
	require.NoError(t, err)
	cronSchedule := "0 0 0 1 1 *"

	// Used in OCR tests
	var (
		peerIDStr      = "12D3KooWPjceQrSwdWXPyLLeABRXmuqt69Rg3sBYbU1Nft9HyQ6X"
		ocrKeyBundleID = "f5bf259689b26f1374efb3c9a9868796953a0f814bb2d39b968d0e61b58620a5"
	)
	p2pPeerID, err := peer.Decode(peerIDStr)
	require.NoError(t, err)
	peerID := p2pkey.PeerID(p2pPeerID)
	ocrKeyID := models.MustSha256HashFromHex(ocrKeyBundleID)
	transmitterAddress, err := ethkey.NewEIP55Address("0x27548a32b9aD5D64c5945EaE9Da5337bc3169D15")
	require.NoError(t, err)

	// Used in keeper test
	fromAddress, err := ethkey.NewEIP55Address("0xa8037A20989AFcBC51798de9762b351D63ff462e")
	require.NoError(t, err)

	testCases := []struct {
		name string
		job  job.Job
		want string
	}{
		{
			name: "direct request spec",
			job: job.Job{
				ID: 1,
				DirectRequestSpec: &job.DirectRequestSpec{
					ContractAddress: contractAddress,
					CreatedAt:       timestamp,
					UpdatedAt:       timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0EEC7E1D-D0D2-476C-A1A8-72DFB6633F46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: `ds1 [type=http method=GET url="https://pricesource1.com"`,
				},
				Type:            job.Type("directrequest"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
			},
			want: fmt.Sprintf(`
			{
				"data":{
					"type":"jobs",
					"id":"1",
					"attributes":{
						"name": "test",
						"schemaVersion": 1,
						"type": "directrequest",
						"maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
						"pipelineSpec": {
							"id": 1,
							"dotDagSource": "ds1 [type=http method=GET url=\"https://pricesource1.com\"",
							"jobID": 0
						},
						"directRequestSpec": {
							"contractAddress": "%s",
							"minIncomingConfirmations": null,
							"minContractPaymentLinkJuels": null,
							"requesters": null,
							"initiator": "runlog",
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						},
						"offChainReportingOracleSpec": null,
						"fluxMonitorSpec": null,
						"keeperSpec": null,
                        "cronSpec": null,
                        "vrfSpec": null,
						"webhookSpec": null,
						"errors": []
					}
				}
			}`, contractAddress),
		},
		{
			name: "fluxmonitor spec",
			job: job.Job{
				ID: 1,
				FluxMonitorSpec: &job.FluxMonitorSpec{
					ContractAddress:   contractAddress,
					Threshold:         0.5,
					IdleTimerPeriod:   1 * time.Minute,
					IdleTimerDisabled: false,
					PollTimerPeriod:   1 * time.Second,
					PollTimerDisabled: false,
					MinPayment:        assets.NewLinkFromJuels(1),
					CreatedAt:         timestamp,
					UpdatedAt:         timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0EEC7E1D-D0D2-476C-A1A8-72DFB6633F46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: `ds1 [type=http method=GET url="https://pricesource1.com"`,
				},
				Type:            job.Type("fluxmonitor"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
			},
			want: fmt.Sprintf(`
			{
				"data":{
					"type":"jobs",
					"id":"1",
					"attributes":{
						"name": "test",
						"schemaVersion": 1,
						"type": "fluxmonitor",
						"maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
						"pipelineSpec": {
							"id": 1,
							"dotDagSource": "ds1 [type=http method=GET url=\"https://pricesource1.com\"",
							"jobID": 0
						},
						"fluxMonitorSpec": {
							"contractAddress": "%s",
							"threshold": 0.5,
							"absoluteThreshold": 0,
							"idleTimerPeriod": "1m0s",
							"idleTimerDisabled": false,
							"pollTimerPeriod": "1s",
							"pollTimerDisabled": false,
              "drumbeatEnabled": false,
              "drumbeatRandomDelay": null,
              "drumbeatSchedule": null,
							"minPayment": "1",
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						},
						"offChainReportingOracleSpec": null,
						"directRequestSpec": null,
						"keeperSpec": null,
                        "cronSpec": null,
                        "vrfSpec": null,
						"webhookSpec": null,
						"errors": []
					}
				}
			}`, contractAddress),
		},
		{
			name: "ocr spec",
			job: job.Job{
				ID: 1,
				OffchainreportingOracleSpec: &job.OffchainReportingOracleSpec{
					ContractAddress:                        contractAddress,
					P2PPeerID:                              &peerID,
					P2PBootstrapPeers:                      pq.StringArray{"/dns4/chain.link/tcp/1234/p2p/xxx"},
					IsBootstrapPeer:                        true,
					EncryptedOCRKeyBundleID:                &ocrKeyID,
					TransmitterAddress:                     &transmitterAddress,
					ObservationTimeout:                     models.Interval(1 * time.Minute),
					BlockchainTimeout:                      models.Interval(1 * time.Minute),
					ContractConfigTrackerSubscribeInterval: models.Interval(1 * time.Minute),
					ContractConfigTrackerPollInterval:      models.Interval(1 * time.Minute),
					ContractConfigConfirmations:            1,
					CreatedAt:                              timestamp,
					UpdatedAt:                              timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0EEC7E1D-D0D2-476C-A1A8-72DFB6633F46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: `ds1 [type=http method=GET url="https://pricesource1.com"`,
				},
				Type:            job.Type("offchainreporting"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
			},
			want: fmt.Sprintf(`
			{
				"data":{
					"type":"jobs",
					"id":"1",
					"attributes":{
						"name": "test",
						"schemaVersion": 1,
						"type": "offchainreporting",
						"maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
						"pipelineSpec": {
							"id": 1,
							"dotDagSource": "ds1 [type=http method=GET url=\"https://pricesource1.com\"",
							"jobID": 0
						},
						"offChainReportingOracleSpec": {
							"contractAddress": "%s",
							"p2pPeerID": "p2p_%s",
							"p2pBootstrapPeers": ["/dns4/chain.link/tcp/1234/p2p/xxx"],
							"isBootstrapPeer": true,
							"keyBundleID": "%s",
							"transmitterAddress": "%s",
							"observationTimeout": "1m0s",
							"blockchainTimeout": "1m0s",
							"contractConfigTrackerSubscribeInterval": "1m0s",
							"contractConfigTrackerPollInterval": "1m0s",
							"contractConfigConfirmations": 1,
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						},
						"fluxMonitorSpec": null,
						"directRequestSpec": null,
						"keeperSpec": null,
                        "cronSpec": null,
                        "vrfSpec": null,
						"webhookSpec": null,
						"errors": []
					}
				}
			}`, contractAddress, peerIDStr, ocrKeyBundleID, transmitterAddress),
		},
		{
			name: "keeper spec",
			job: job.Job{
				ID: 1,
				KeeperSpec: &job.KeeperSpec{
					ContractAddress: contractAddress,
					FromAddress:     fromAddress,
					CreatedAt:       timestamp,
					UpdatedAt:       timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0EEC7E1D-D0D2-476C-A1A8-72DFB6633F46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: "",
				},
				Type:            job.Type("keeper"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
			},
			want: fmt.Sprintf(`
			{
				"data":{
					"type":"jobs",
					"id":"1",
					"attributes":{
						"name": "test",
						"schemaVersion": 1,
						"type": "keeper",
						"maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
						"pipelineSpec": {
							"id": 1,
							"dotDagSource": "",
							"jobID": 0
						},
						"keeperSpec": {
							"contractAddress": "%s",
							"fromAddress": "%s",
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						},
						"fluxMonitorSpec": null,
						"directRequestSpec": null,
						"cronSpec": null,
						"webhookSpec": null,
						"offChainReportingOracleSpec": null,
                        "cronSpec": null,
                        "vrfSpec": null,
						"errors": []
					}
				}
			}`, contractAddress, fromAddress),
		},
		{
			name: "cron spec",
			job: job.Job{
				ID: 1,
				CronSpec: &job.CronSpec{
					CronSchedule: cronSchedule,
					CreatedAt:    timestamp,
					UpdatedAt:    timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0EEC7E1D-D0D2-476C-A1A8-72DFB6633F46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: "",
				},
				Type:            job.Type("cron"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
			},
			want: fmt.Sprintf(`
            {
                "data":{
                    "type":"jobs",
                    "id":"1",
                    "attributes":{
                        "name": "test",
                        "schemaVersion": 1,
                        "type": "cron",
                        "maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
                        "pipelineSpec": {
                            "id": 1,
                            "dotDagSource": "",
														"jobID": 0
                        },
                        "cronSpec": {
                            "schedule": "%s",
                            "createdAt":"2000-01-01T00:00:00Z",
                            "updatedAt":"2000-01-01T00:00:00Z"
                        },
                        "fluxMonitorSpec": null,
                        "directRequestSpec": null,
                        "keeperSpec": null,
                        "offChainReportingOracleSpec": null,
						"vrfSpec": null,
                        "webhookSpec": null,
                        "errors": []
                    }
                }
            }`, cronSchedule),
		},
		{
			name: "webhook spec",
			job: job.Job{
				ID: 1,
				WebhookSpec: &job.WebhookSpec{
					CreatedAt: timestamp,
					UpdatedAt: timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0eec7e1d-d0d2-476c-a1a8-72dfb6633f46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: "",
				},
				Type:            job.Type("webhook"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
			},
			want: `
			{
				"data":{
					"type":"jobs",
					"id":"1",
					"attributes":{
						"name": "test",
						"schemaVersion": 1,
						"type": "webhook",
						"maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
						"pipelineSpec": {
							"id": 1,
							"dotDagSource": "",
							"jobID": 0
						},
						"webhookSpec": {
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						},
						"fluxMonitorSpec": null,
						"directRequestSpec": null,
						"keeperSpec": null,
						"cronSpec": null,
						"offChainReportingOracleSpec": null,
                        "vrfSpec": null,
						"errors": []
					}
				}
			}`,
		},
		{
			name: "with errors",
			job: job.Job{
				ID: 1,
				KeeperSpec: &job.KeeperSpec{
					ContractAddress: contractAddress,
					FromAddress:     fromAddress,
					CreatedAt:       timestamp,
					UpdatedAt:       timestamp,
				},
				ExternalJobID: uuid.FromStringOrNil("0EEC7E1D-D0D2-476C-A1A8-72DFB6633F46"),
				PipelineSpec: &pipeline.Spec{
					ID:           1,
					DotDagSource: "",
				},
				Type:            job.Type("keeper"),
				SchemaVersion:   1,
				Name:            null.StringFrom("test"),
				MaxTaskDuration: models.Interval(1 * time.Minute),
				JobSpecErrors: []job.SpecError{
					{
						ID:          200,
						JobID:       1,
						Description: "some error",
						Occurrences: 1,
						CreatedAt:   timestamp,
						UpdatedAt:   timestamp,
					},
				},
			},
			want: fmt.Sprintf(`
			{
				"data":{
					"type":"jobs",
					"id":"1",
					"attributes":{
						"name": "test",
						"schemaVersion": 1,
						"type": "keeper",
						"maxTaskDuration": "1m0s",
					    "externalJobID":"0eec7e1d-d0d2-476c-a1a8-72dfb6633f46",
						"pipelineSpec": {
							"id": 1,
							"dotDagSource": "",
							"jobID": 0
						},
						"keeperSpec": {
							"contractAddress": "%s",
							"fromAddress": "%s",
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						},
						"fluxMonitorSpec": null,
						"directRequestSpec": null,
						"cronSpec": null,
						"webhookSpec": null,
						"offChainReportingOracleSpec": null,
						"vrfSpec": null,
						"errors": [{
							"id": 200,
							"description": "some error",
							"occurrences": 1,
							"createdAt":"2000-01-01T00:00:00Z",
							"updatedAt":"2000-01-01T00:00:00Z"
						}]
					}
				}
			}`, contractAddress, fromAddress),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := presenters.NewJobResource(tc.job)
			b, err := jsonapi.Marshal(r)
			require.NoError(t, err)

			assert.JSONEq(t, tc.want, string(b))
		})
	}
}
