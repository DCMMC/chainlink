package presenters

import (
	"github.com/DCMMC/chainlink/core/logger"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/vrfkey"
)

type VRFKeyResource struct {
	JAID
	Compressed   string `json:"compressed"`
	Uncompressed string `json:"uncompressed"`
	Hash         string `json:"hash"`
}

// GetName implements the api2go EntityNamer interface
func (VRFKeyResource) GetName() string {
	return "encryptedVRFKeys"
}

func NewVRFKeyResource(key vrfkey.KeyV2) *VRFKeyResource {
	uncompressed, err := key.PublicKey.StringUncompressed()
	if err != nil {
		logger.Error("unable to get uncompressed pk", "err", err)
	}
	return &VRFKeyResource{
		JAID:         NewJAID(key.PublicKey.String()),
		Compressed:   key.PublicKey.String(),
		Uncompressed: uncompressed,
		Hash:         key.PublicKey.MustHash().String(),
	}
}

func NewVRFKeyResources(keys []vrfkey.KeyV2) []VRFKeyResource {
	rs := []VRFKeyResource{}
	for _, key := range keys {
		rs = append(rs, *NewVRFKeyResource(key))
	}

	return rs
}
