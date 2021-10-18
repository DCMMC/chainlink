package vrf

import (
	"github.com/DCMMC/chainlink/core/services/keystore/keys/vrfkey"
	"github.com/DCMMC/chainlink/core/services/vrf/proof"
)

func GenerateProofResponseFromProof(p vrfkey.Proof, s proof.PreSeedData) (
	proof.MarshaledOnChainResponse, error) {
	return proof.GenerateProofResponseFromProof(p, s)
}
