package keystore_test

import (
	"fmt"
	"testing"

	"github.com/DCMMC/chainlink/core/internal/cltest"
	"github.com/DCMMC/chainlink/core/internal/testutils/pgtest"
	"github.com/DCMMC/chainlink/core/services/keystore"
	"github.com/DCMMC/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/DCMMC/chainlink/core/services/offchainreporting"
	"github.com/stretchr/testify/require"
)

func Test_P2PKeyStore_E2E(t *testing.T) {
	db := pgtest.NewGormDB(t)
	keyStore := keystore.ExposedNewMaster(t, db)
	keyStore.Unlock(cltest.Password)
	ks := keyStore.P2P()
	reset := func() {
		require.NoError(t, db.Exec("DELETE FROM encrypted_key_rings").Error)
		keyStore.ResetXXXTestOnly()
		keyStore.Unlock(cltest.Password)
	}

	t.Run("initializes with an empty state", func(t *testing.T) {
		defer reset()
		keys, err := ks.GetAll()
		require.NoError(t, err)
		require.Equal(t, 0, len(keys))
	})

	t.Run("errors when getting non-existant ID", func(t *testing.T) {
		defer reset()
		_, err := ks.Get("non-existant-id")
		require.Error(t, err)
	})

	t.Run("creates a key", func(t *testing.T) {
		defer reset()
		key, err := ks.Create()
		require.NoError(t, err)
		retrievedKey, err := ks.Get(key.ID())
		require.NoError(t, err)
		require.Equal(t, key, retrievedKey)
	})

	t.Run("imports and exports a key", func(t *testing.T) {
		defer reset()
		key, err := ks.Create()
		require.NoError(t, err)
		exportJSON, err := ks.Export(key.ID(), cltest.Password)
		require.NoError(t, err)
		_, err = ks.Delete(key.ID())
		require.NoError(t, err)
		_, err = ks.Get(key.ID())
		require.Error(t, err)
		importedKey, err := ks.Import(exportJSON, cltest.Password)
		require.NoError(t, err)
		require.Equal(t, key.ID(), importedKey.ID())
		retrievedKey, err := ks.Get(key.ID())
		require.NoError(t, err)
		require.Equal(t, importedKey, retrievedKey)
	})

	t.Run("adds an externally created key / deletes a key", func(t *testing.T) {
		defer reset()
		newKey, err := p2pkey.NewV2()
		require.NoError(t, err)
		err = ks.Add(newKey)
		require.NoError(t, err)
		keys, err := ks.GetAll()
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		_, err = ks.Delete(newKey.ID())
		require.NoError(t, err)
		keys, err = ks.GetAll()
		require.NoError(t, err)
		require.Equal(t, 0, len(keys))
		_, err = ks.Get(newKey.ID())
		require.Error(t, err)
	})

	t.Run("ensures key", func(t *testing.T) {
		defer reset()
		_, didExist, err := ks.EnsureKey()
		require.NoError(t, err)
		require.False(t, didExist)
		_, didExist, err = ks.EnsureKey()
		require.NoError(t, err)
		require.True(t, didExist)
		keys, err := ks.GetAll()
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
	})

	t.Run("GetOrFirst", func(t *testing.T) {
		defer reset()
		_, err := ks.GetOrFirst("")
		require.Contains(t, err.Error(), "no p2p keys exist")
		id := cltest.DefaultP2PPeerID.Raw()
		_, err = ks.GetOrFirst(id)
		require.Contains(t, err.Error(), fmt.Sprintf("unable to find P2P key with id %s", id))
		k1, err := ks.Create()
		require.NoError(t, err)
		k2, err := ks.GetOrFirst("")
		require.NoError(t, err)
		require.Equal(t, k1, k2)
		k3, err := ks.GetOrFirst(k1.ID())
		require.NoError(t, err)
		require.Equal(t, k1, k3)
		_, err = ks.Create()
		require.NoError(t, err)
		_, err = ks.GetOrFirst("")
		require.Contains(t, err.Error(), "multiple p2p keys found")
		k4, err := ks.GetOrFirst(k1.ID())
		require.NoError(t, err)
		require.Equal(t, k1, k4)
	})

	t.Run("clears p2p_peers on delete", func(t *testing.T) {
		key, err := ks.Create()
		require.NoError(t, err)
		p2pPeer1 := offchainreporting.P2PPeer{
			Addr:   cltest.NewAddress().Hex(),
			PeerID: cltest.DefaultPeerID, // different p2p key
		}
		p2pPeer2 := offchainreporting.P2PPeer{
			Addr:   cltest.NewAddress().Hex(),
			PeerID: key.PeerID().Raw(),
		}
		require.NoError(t, db.Create(&p2pPeer1).Error)
		require.NoError(t, db.Create(&p2pPeer2).Error)
		cltest.AssertCount(t, db, offchainreporting.P2PPeer{}, 2)
		ks.Delete(key.ID())
		cltest.AssertCount(t, db, offchainreporting.P2PPeer{}, 1)
	})

	t.Run("imports a key exported from a v1 keystore", func(t *testing.T) {
		exportedKey := `{"publicKey":"fcc1fdebde28322dde17233fe7bd6dcde447d60d5cc1de518962deed102eea35","peerID":"p2p_12D3KooWSq2UZgSXvhGLG5uuAAmz1JNjxHMJViJB39aorvbbYo8p","crypto":{"cipher":"aes-128-ctr","ciphertext":"adb2dff72148a8cd467f6f06a03869e7cedf180cf2a4decdb86875b2e1cf3e58c4bd2b721ecdaa88a0825fa9abfc309bf32dbb35a5c0b6cb01ac89a956d78e0550eff351","cipherparams":{"iv":"6cc4381766a4efc39f762b2b8d09dfba"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"ff5055ae4cdcdc2d0404307d578262e2caeb0210f82db3a0ecbdba727c6f5259"},"mac":"d37e4f1dea98d85960ef3205099fc71741715ae56a3b1a8f9215a78de9b95595"}}`
		importedKey, err := ks.Import([]byte(exportedKey), cltest.Password)
		require.NoError(t, err)
		require.Equal(t, "12D3KooWSq2UZgSXvhGLG5uuAAmz1JNjxHMJViJB39aorvbbYo8p", importedKey.ID())
	})
}
