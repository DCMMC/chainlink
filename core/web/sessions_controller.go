package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.uber.org/multierr"

	"github.com/DCMMC/chainlink/core/logger"
	"github.com/DCMMC/chainlink/core/services/chainlink"
	clsessions "github.com/DCMMC/chainlink/core/sessions"
)

// SessionsController manages session requests.
type SessionsController struct {
	App      chainlink.Application
	Sessions *clsessions.WebAuthnSessionStore
}

// Create creates a session ID for the given user credentials, and returns it
// in a cookie.
func (sc *SessionsController) Create(c *gin.Context) {
	defer sc.App.WakeSessionReaper()
	logger.Debugf("TRACE: Starting Session Creation")
	if sc.Sessions == nil {
		sc.Sessions = clsessions.NewWebAuthnSessionStore()
	}

	session := sessions.Default(c)
	var sr clsessions.SessionRequest
	if err := c.ShouldBindJSON(&sr); err != nil {
		jsonAPIError(c, http.StatusBadRequest, fmt.Errorf("error binding json %v", err))
		return
	}

	// Does this user have 2FA enabled?
	userWebAuthnTokens, err := sc.App.SessionORM().GetUserWebAuthn(sr.Email)
	if err != nil {
		logger.Errorf("Error loading user WebAuthn data: %s", err)
		jsonAPIError(c, http.StatusInternalServerError, errors.New("Internal Server Error"))
		return
	}

	// If the user has registered MFA tokens, then populate our session store and context
	// required for successful WebAuthn authentication
	if len(userWebAuthnTokens) > 0 {
		sr.SessionStore = sc.Sessions
		sr.RequestContext = c
		sr.WebAuthnConfig = sc.App.GetWebAuthnConfiguration()
	}

	sid, err := sc.App.SessionORM().CreateSession(sr)
	if err != nil {
		jsonAPIError(c, http.StatusUnauthorized, err)
		return
	}

	if err := saveSessionID(session, sid); err != nil {
		jsonAPIError(c, http.StatusInternalServerError, multierr.Append(errors.New("unable to save session id"), err))
		return
	}

	jsonAPIResponse(c, Session{Authenticated: true}, "session")
}

// Destroy erases the session ID for the sole API user.
func (sc *SessionsController) Destroy(c *gin.Context) {
	defer sc.App.WakeSessionReaper()

	session := sessions.Default(c)
	defer session.Clear()
	sessionID, ok := session.Get(SessionIDKey).(string)
	if !ok {
		jsonAPIResponse(c, Session{Authenticated: false}, "session")
		return
	}
	if err := sc.App.SessionORM().DeleteUserSession(sessionID); err != nil {
		jsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	jsonAPIResponse(c, Session{Authenticated: false}, "session")
}

func saveSessionID(session sessions.Session, sessionID string) error {
	session.Set(SessionIDKey, sessionID)
	return session.Save()
}

type Session struct {
	Authenticated bool `json:"authenticated"`
}

// GetID returns the jsonapi ID.
func (s Session) GetID() string {
	return "sessionID"
}

// GetName returns the collection name for jsonapi.
func (Session) GetName() string {
	return "session"
}

// SetID is used to conform to the UnmarshallIdentifier interface for
// deserializing from jsonapi documents.
func (*Session) SetID(string) error {
	return nil
}
