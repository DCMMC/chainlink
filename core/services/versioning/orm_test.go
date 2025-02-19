package versioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DCMMC/chainlink/core/internal/testutils/pgtest"
	"github.com/DCMMC/chainlink/core/logger"
)

func TestORM_NodeVersion_UpsertNodeVersion(t *testing.T) {
	db := pgtest.NewSqlxDB(t)
	orm := NewORM(db, logger.TestLogger(t))

	err := orm.UpsertNodeVersion(NewNodeVersion("9.9.8"))
	require.NoError(t, err)

	ver, err := orm.FindLatestNodeVersion()

	require.NoError(t, err)
	require.NotNil(t, ver)
	require.Equal(t, "9.9.8", ver.Version)
	require.NotZero(t, ver.CreatedAt)

	// Testing Upsert
	require.NoError(t, orm.UpsertNodeVersion(NewNodeVersion("9.9.8")))

	err = orm.UpsertNodeVersion(NewNodeVersion("9.9.7"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Application version (9.9.7) is older than database version (9.9.8). Only Chainlink 9.9.8 or later can be run on this database")

	require.NoError(t, orm.UpsertNodeVersion(NewNodeVersion("9.9.9")))

	var count int
	err = db.QueryRowx(`SELECT count(*) FROM node_versions`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	ver, err = orm.FindLatestNodeVersion()

	require.NoError(t, err)
	require.NotNil(t, ver)
	require.Equal(t, "9.9.9", ver.Version)

	// invalid semver returns error
	err = orm.UpsertNodeVersion(NewNodeVersion("random_12345"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "\"random_12345\" is not valid semver: Invalid Semantic Version")

	ver, err = orm.FindLatestNodeVersion()
	require.NoError(t, err)
	require.NotNil(t, ver)
	require.Equal(t, "9.9.9", ver.Version)
}

func Test_Version_CheckVersion(t *testing.T) {
	db := pgtest.NewSqlxDB(t)

	lggr := logger.TestLogger(t)

	orm := NewORM(db, lggr)

	err := orm.UpsertNodeVersion(NewNodeVersion("9.9.8"))
	require.NoError(t, err)

	// invalid app version semver returns error
	err = CheckVersion(db, lggr, "unset")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `Application version "unset" is not valid semver`)
	err = CheckVersion(db, lggr, "some old bollocks")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `Application version "some old bollocks" is not valid semver`)

	// lower version returns error
	err = CheckVersion(db, lggr, "9.9.7")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Application version (9.9.7) is older than database version (9.9.8). Only Chainlink 9.9.8 or later can be run on this database")

	// equal version is ok
	err = CheckVersion(db, lggr, "9.9.8")
	require.NoError(t, err)

	// greater version is ok
	err = CheckVersion(db, lggr, "9.9.9")
	require.NoError(t, err)
}

func TestORM_NodeVersion_FindLatestNodeVersion(t *testing.T) {
	db := pgtest.NewSqlxDB(t)
	orm := NewORM(db, logger.TestLogger(t))

	// Not Found
	_, err := orm.FindLatestNodeVersion()
	require.Error(t, err)

	err = orm.UpsertNodeVersion(NewNodeVersion("9.9.8"))
	require.NoError(t, err)

	ver, err := orm.FindLatestNodeVersion()

	require.NoError(t, err)
	require.NotNil(t, ver)
	require.Equal(t, "9.9.8", ver.Version)
	require.NotZero(t, ver.CreatedAt)
}
