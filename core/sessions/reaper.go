package sessions

import (
	"database/sql"
	"time"

	"github.com/DCMMC/chainlink/core/logger"
	"github.com/DCMMC/chainlink/core/store/models"
	"github.com/DCMMC/chainlink/core/utils"
)

type sessionReaper struct {
	db     *sql.DB
	config SessionReaperConfig
}

type SessionReaperConfig interface {
	SessionTimeout() models.Duration
	ReaperExpiration() models.Duration
}

// NewSessionReaper creates a reaper that cleans stale sessions from the store.
func NewSessionReaper(db *sql.DB, config SessionReaperConfig) utils.SleeperTask {
	return utils.NewSleeperTask(&sessionReaper{
		db,
		config,
	})
}

func (sr *sessionReaper) Work() {
	recordCreationStaleThreshold := sr.config.ReaperExpiration().Before(
		sr.config.SessionTimeout().Before(time.Now()))
	err := sr.deleteStaleSessions(recordCreationStaleThreshold)
	if err != nil {
		logger.Error("unable to reap stale sessions: ", err)
	}
}

// DeleteStaleSessions deletes all sessions before the passed time.
func (sr *sessionReaper) deleteStaleSessions(before time.Time) error {
	_, err := sr.db.Exec("DELETE FROM sessions WHERE last_used < $1", before)
	return err
}
