package ethkey

import (
	"time"

	"github.com/DCMMC/chainlink/core/utils"
)

type State struct {
	ID         int32 `gorm:"primary_key"`
	Address    EIP55Address
	NextNonce  int64
	IsFunding  bool
	EVMChainID utils.Big `gorm:"column:evm_chain_id"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	lastUsed   time.Time
}

func (State) TableName() string {
	return "eth_key_states"
}

func (s State) KeyID() string {
	return s.Address.Hex()
}

// lastUsed is an internal field and ought not be persisted to the database or
// exposed outside of the application
func (s State) LastUsed() time.Time {
	return s.lastUsed
}

func (s *State) WasUsed() {
	s.lastUsed = time.Now()
}
