package keystore

import (
	"math/big"
	"testing"

	"github.com/DCMMC/chainlink/core/services/keystore/keys/csakey"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/ethkey"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/ocrkey"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/vrfkey"
	"github.com/DCMMC/chainlink/core/utils"
	"github.com/stretchr/testify/require"
)

const password = "password"

func TestKeyRing_Encrypt_Decrypt(t *testing.T) {
	csa1, csa2 := csakey.MustNewV2XXXTestingOnly(big.NewInt(1)), csakey.MustNewV2XXXTestingOnly(big.NewInt(2))
	eth1, eth2 := mustNewEthKey(t), mustNewEthKey(t)
	ocr1, ocr2 := ocrkey.MustNewV2XXXTestingOnly(big.NewInt(1)), ocrkey.MustNewV2XXXTestingOnly(big.NewInt(2))
	p2p1, p2p2 := p2pkey.MustNewV2XXXTestingOnly(big.NewInt(1)), p2pkey.MustNewV2XXXTestingOnly(big.NewInt(2))
	vrf1, vrf2 := vrfkey.MustNewV2XXXTestingOnly(big.NewInt(1)), vrfkey.MustNewV2XXXTestingOnly(big.NewInt(2))
	originalKeyRingRaw := rawKeyRing{
		CSA: []csakey.Raw{csa1.Raw(), csa2.Raw()},
		Eth: []ethkey.Raw{eth1.Raw(), eth2.Raw()},
		OCR: []ocrkey.Raw{ocr1.Raw(), ocr2.Raw()},
		P2P: []p2pkey.Raw{p2p1.Raw(), p2p2.Raw()},
		VRF: []vrfkey.Raw{vrf1.Raw(), vrf2.Raw()},
	}
	originalKeyRing, err := originalKeyRingRaw.keys()
	require.NoError(t, err)

	encryptedKeyRing, err := originalKeyRing.Encrypt(password, utils.FastScryptParams)
	require.NoError(t, err)
	decryptedKeyRing, err := encryptedKeyRing.Decrypt(password)
	require.NoError(t, err)
	// compare csa keys
	require.Equal(t, 2, len(decryptedKeyRing.CSA))
	require.Equal(t, originalKeyRing.CSA[csa1.ID()].PublicKey, decryptedKeyRing.CSA[csa1.ID()].PublicKey)
	require.Equal(t, originalKeyRing.CSA[csa2.ID()].PublicKey, decryptedKeyRing.CSA[csa2.ID()].PublicKey)
	// compare eth keys
	require.Equal(t, 2, len(decryptedKeyRing.Eth))
	require.Equal(t, originalKeyRing.Eth[eth1.ID()].Address, decryptedKeyRing.Eth[eth1.ID()].Address)
	require.Equal(t, originalKeyRing.Eth[eth2.ID()].Address, decryptedKeyRing.Eth[eth2.ID()].Address)
	// compare ocr keys
	require.Equal(t, 2, len(decryptedKeyRing.OCR))
	require.Equal(t, originalKeyRing.OCR[ocr1.ID()].OnChainSigning.X, decryptedKeyRing.OCR[ocr1.ID()].OnChainSigning.X)
	require.Equal(t, originalKeyRing.OCR[ocr1.ID()].OnChainSigning.Y, decryptedKeyRing.OCR[ocr1.ID()].OnChainSigning.Y)
	require.Equal(t, originalKeyRing.OCR[ocr1.ID()].OnChainSigning.D, decryptedKeyRing.OCR[ocr1.ID()].OnChainSigning.D)
	require.Equal(t, originalKeyRing.OCR[ocr1.ID()].OffChainSigning, decryptedKeyRing.OCR[ocr1.ID()].OffChainSigning)
	require.Equal(t, originalKeyRing.OCR[ocr1.ID()].OffChainEncryption, decryptedKeyRing.OCR[ocr1.ID()].OffChainEncryption)
	require.Equal(t, originalKeyRing.OCR[ocr2.ID()].OnChainSigning.X, decryptedKeyRing.OCR[ocr2.ID()].OnChainSigning.X)
	require.Equal(t, originalKeyRing.OCR[ocr2.ID()].OnChainSigning.Y, decryptedKeyRing.OCR[ocr2.ID()].OnChainSigning.Y)
	require.Equal(t, originalKeyRing.OCR[ocr2.ID()].OnChainSigning.D, decryptedKeyRing.OCR[ocr2.ID()].OnChainSigning.D)
	require.Equal(t, originalKeyRing.OCR[ocr2.ID()].OffChainSigning, decryptedKeyRing.OCR[ocr2.ID()].OffChainSigning)
	require.Equal(t, originalKeyRing.OCR[ocr2.ID()].OffChainEncryption, decryptedKeyRing.OCR[ocr2.ID()].OffChainEncryption)
	// compare p2p keys
	require.Equal(t, 2, len(decryptedKeyRing.P2P))
	require.Equal(t, originalKeyRing.P2P[p2p1.ID()].GetPublic(), decryptedKeyRing.P2P[p2p1.ID()].GetPublic())
	require.Equal(t, originalKeyRing.P2P[p2p1.ID()].PeerID(), decryptedKeyRing.P2P[p2p1.ID()].PeerID())
	require.Equal(t, originalKeyRing.P2P[p2p2.ID()].GetPublic(), decryptedKeyRing.P2P[p2p2.ID()].GetPublic())
	require.Equal(t, originalKeyRing.P2P[p2p2.ID()].PeerID(), decryptedKeyRing.P2P[p2p2.ID()].PeerID())
	// compare vrf keys
	require.Equal(t, 2, len(decryptedKeyRing.VRF))
	require.Equal(t, originalKeyRing.VRF[vrf1.ID()].PublicKey, decryptedKeyRing.VRF[vrf1.ID()].PublicKey)
	require.Equal(t, originalKeyRing.VRF[vrf2.ID()].PublicKey, decryptedKeyRing.VRF[vrf2.ID()].PublicKey)
}
