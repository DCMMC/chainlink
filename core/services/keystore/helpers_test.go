package keystore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/DCMMC/chainlink/core/logger"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/ethkey"
	"github.com/DCMMC/chainlink/core/utils"
)

func mustNewEthKey(t *testing.T) *ethkey.KeyV2 {
	key, err := ethkey.NewV2()
	require.NoError(t, err)
	return &key
}

type ExportedEncryptedKeyRing = encryptedKeyRing

func ExposedNewMaster(t *testing.T, db *gorm.DB) *master {
	return newMaster(db, utils.FastScryptParams, logger.TestLogger(t))
}

func (m *master) ExportedSave() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.save()
}

func (m *master) ResetXXXTestOnly() {
	m.keyRing = newKeyRing()
	m.keyStates = newKeyStates()
	m.password = ""
}
