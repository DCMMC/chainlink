package fluxmonitorv2_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"

	"gorm.io/gorm"

	"github.com/DCMMC/chainlink/core/internal/cltest"
	"github.com/DCMMC/chainlink/core/internal/mocks"
	"github.com/DCMMC/chainlink/core/services/fluxmonitorv2"
	fmmocks "github.com/DCMMC/chainlink/core/services/fluxmonitorv2/mocks"
	"github.com/stretchr/testify/assert"
)

func TestFluxAggregatorContractSubmitter_Submit(t *testing.T) {
	var (
		fluxAggregator = new(mocks.FluxAggregator)
		orm            = new(fmmocks.ORM)
		keyStore       = new(fmmocks.KeyStoreInterface)
		gasLimit       = uint64(2100)
		submitter      = fluxmonitorv2.NewFluxAggregatorContractSubmitter(fluxAggregator, orm, keyStore, gasLimit)

		toAddress   = cltest.NewAddress()
		fromAddress = cltest.NewAddress()
		roundID     = big.NewInt(1)
		submission  = big.NewInt(2)
	)

	payload, err := fluxmonitorv2.FluxAggregatorABI.Pack("submit", roundID, submission)
	assert.NoError(t, err)

	keyStore.On("GetRoundRobinAddress", mock.Anything).Return(fromAddress, nil)
	fluxAggregator.On("Address").Return(toAddress)
	orm.On("CreateEthTransaction", mock.Anything, fromAddress, toAddress, payload, gasLimit).Return(nil)

	err = submitter.Submit(&gorm.DB{}, roundID, submission)
	assert.NoError(t, err)
}
