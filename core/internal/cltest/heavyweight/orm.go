package heavyweight

// The heavyweight package contains cltest items that are costly and you should
// think **real carefully** before using in your tests.

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/DCMMC/chainlink/core/internal/cltest"
	"github.com/DCMMC/chainlink/core/internal/testutils/configtest"
	"github.com/DCMMC/chainlink/core/services/postgres"
	"github.com/DCMMC/chainlink/core/store/dialects"
	migrations "github.com/DCMMC/chainlink/core/store/migrate"
	"github.com/smartcontractkit/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

// FullTestDB creates an DB which runs in a separate database than the normal
// unit tests, so you can do things like use other Postgres connection types
// with it.
func FullTestDB(t *testing.T, name string, migrate bool, loadFixtures bool) (*configtest.TestGeneralConfig, *sqlx.DB, *gorm.DB) {
	overrides := configtest.GeneralConfigOverrides{
		SecretGenerator: cltest.MockSecretGenerator{},
	}
	gcfg := configtest.NewTestGeneralConfigWithOverrides(t, overrides)
	gcfg.SetDialect(dialects.Postgres)

	require.NoError(t, os.MkdirAll(gcfg.RootDir(), 0700))
	migrationTestDBURL, err := dropAndCreateThrowawayTestDB(gcfg.DatabaseURL(), name)
	require.NoError(t, err)
	db, gormDB, err := postgres.NewConnection(migrationTestDBURL, string(dialects.Postgres), postgres.Config{
		LogSQLStatements: gcfg.LogSQLStatements(),
		MaxOpenConns:     gcfg.ORMMaxOpenConns(),
		MaxIdleConns:     gcfg.ORMMaxIdleConns(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
		os.RemoveAll(gcfg.RootDir())
	})
	postgres.SetLogAllQueries(gormDB, gcfg.LogSQLMigrations())
	gcfg.Overrides.DatabaseURL = null.StringFrom(migrationTestDBURL)
	if migrate {
		require.NoError(t, migrations.Migrate(db.DB))
	}
	if loadFixtures {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("could not get runtime.Caller(0)")
		}
		filepath := path.Join(path.Dir(filename), "../../../store/fixtures/fixtures.sql")
		fixturesSQL, err := ioutil.ReadFile(filepath)
		require.NoError(t, err)
		_, err = db.Exec(string(fixturesSQL))
		require.NoError(t, err)
	}
	postgres.SetLogAllQueries(gormDB, gcfg.LogSQLStatements())

	return gcfg, db, gormDB
}

func dropAndCreateThrowawayTestDB(parsed url.URL, postfix string) (string, error) {
	if parsed.Path == "" {
		return "", errors.New("path missing from database URL")
	}

	dbname := fmt.Sprintf("%s_%s", parsed.Path[1:], postfix)
	if len(dbname) > 62 {
		return "", fmt.Errorf("dbname %v too long, max is 63 bytes. Try a shorter postfix", dbname)
	}
	// Cannot drop test database if we are connected to it, so we must connect
	// to a different one. 'postgres' should be present on all postgres installations
	parsed.Path = "/postgres"
	db, err := sql.Open(string(dialects.Postgres), parsed.String())
	if err != nil {
		return "", fmt.Errorf("unable to open postgres database for creating test db: %+v", err)
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbname))
	if err != nil {
		return "", fmt.Errorf("unable to drop postgres migrations test database: %v", err)
	}
	// `CREATE DATABASE $1` does not seem to work w CREATE DATABASE
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
	if err != nil {
		return "", fmt.Errorf("unable to create postgres migrations test database with name '%s': %v", dbname, err)
	}
	parsed.Path = fmt.Sprintf("/%s", dbname)
	return parsed.String(), nil
}
