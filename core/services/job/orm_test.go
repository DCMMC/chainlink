package job

import (
	"testing"

	"gorm.io/gorm"

	"github.com/DCMMC/chainlink/core/chains/evm"
	"github.com/DCMMC/chainlink/core/logger"
	"github.com/DCMMC/chainlink/core/services/keystore"
	"github.com/DCMMC/chainlink/core/services/pipeline"
)

func NewTestORM(t *testing.T, db *gorm.DB,
	chainSet evm.ChainSet,
	pipelineORM pipeline.ORM,
	keyStore keystore.Master) ORM {
	o := NewORM(db, chainSet, pipelineORM, keyStore, logger.TestLogger(t))
	t.Cleanup(func() { o.Close() })
	return o
}
