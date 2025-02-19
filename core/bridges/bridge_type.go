package bridges

import (
	"crypto/subtle"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/DCMMC/chainlink/core/assets"
	"github.com/DCMMC/chainlink/core/store/models"
	"github.com/DCMMC/chainlink/core/utils"
)

// BridgeTypeRequest is the incoming record used to create a BridgeType
type BridgeTypeRequest struct {
	Name                   TaskType      `json:"name"`
	URL                    models.WebURL `json:"url"`
	Confirmations          uint32        `json:"confirmations"`
	MinimumContractPayment *assets.Link  `json:"minimumContractPayment"`
}

// GetID returns the ID of this structure for jsonapi serialization.
func (bt BridgeTypeRequest) GetID() string {
	return bt.Name.String()
}

// GetName returns the pluralized "type" of this structure for jsonapi serialization.
func (bt BridgeTypeRequest) GetName() string {
	return "bridges"
}

// SetID is used to set the ID of this structure when deserializing from jsonapi documents.
func (bt *BridgeTypeRequest) SetID(value string) error {
	name, err := NewTaskType(value)
	bt.Name = name
	return err
}

// BridgeTypeAuthentication is the record returned in response to a request to create a BridgeType
type BridgeTypeAuthentication struct {
	Name                   TaskType
	URL                    models.WebURL
	Confirmations          uint32
	IncomingToken          string
	OutgoingToken          string
	MinimumContractPayment *assets.Link
}

// BridgeType is used for external adapters and has fields for
// the name of the adapter and its URL.
type BridgeType struct {
	Name                   TaskType `gorm:"primary_key"`
	URL                    models.WebURL
	Confirmations          uint32
	IncomingTokenHash      string
	Salt                   string
	OutgoingToken          string
	MinimumContractPayment *assets.Link `gorm:"type:varchar(255)"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// NewBridgeType returns a bridge bridge type authentication (with plaintext
// password) and a bridge type (with hashed password, for persisting)
func NewBridgeType(btr *BridgeTypeRequest) (*BridgeTypeAuthentication,
	*BridgeType, error) {
	incomingToken := utils.NewSecret(24)
	outgoingToken := utils.NewSecret(24)
	salt := utils.NewSecret(24)

	hash, err := incomingTokenHash(incomingToken, salt)
	if err != nil {
		return nil, nil, err
	}

	return &BridgeTypeAuthentication{
			Name:                   btr.Name,
			URL:                    btr.URL,
			Confirmations:          btr.Confirmations,
			IncomingToken:          incomingToken,
			OutgoingToken:          outgoingToken,
			MinimumContractPayment: btr.MinimumContractPayment,
		}, &BridgeType{
			Name:                   btr.Name,
			URL:                    btr.URL,
			Confirmations:          btr.Confirmations,
			IncomingTokenHash:      hash,
			Salt:                   salt,
			OutgoingToken:          outgoingToken,
			MinimumContractPayment: btr.MinimumContractPayment,
		}, nil
}

// AuthenticateBridgeType returns true if the passed token matches its
// IncomingToken, or returns false with an error.
func AuthenticateBridgeType(bt *BridgeType, token string) (bool, error) {
	hash, err := incomingTokenHash(token, bt.Salt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(bt.IncomingTokenHash)) == 1, nil
}

func incomingTokenHash(token, salt string) (string, error) {
	input := fmt.Sprintf("%s-%s", token, salt)
	hash, err := utils.Sha256(input)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// NOTE: latestAnswer and updatedAt is the only metadata used.
// Currently market closer adapter and outlier detection depend latestAnswer.
// https://github.com/smartcontractkit/external-adapters-js/tree/f474bd2e2de13ebe5c9dc3df36ebb7018817005e/composite/market-closure
// https://github.com/smartcontractkit/external-adapters-js/tree/5abb8e5ec2024f724fd39122897baa63c3cd0167/composite/outlier-detection
type BridgeMetaData struct {
	LatestAnswer *big.Int `json:"latestAnswer"`
	UpdatedAt    *big.Int `json:"updatedAt"` // A unix timestamp
}

type BridgeMetaDataJSON struct {
	Meta BridgeMetaData
}

func MarshalBridgeMetaData(latestAnswer *big.Int, updatedAt *big.Int) (map[string]interface{}, error) {
	b, err := json.Marshal(&BridgeMetaData{LatestAnswer: latestAnswer, UpdatedAt: updatedAt})
	if err != nil {
		return nil, err
	}
	var mp map[string]interface{}
	err = json.Unmarshal(b, &mp)
	if err != nil {
		return nil, err
	}
	return mp, nil
}

// TaskType defines what Adapter a TaskSpec will use.
type TaskType string

// NewTaskType returns a formatted Task type.
func NewTaskType(val string) (TaskType, error) {
	re := regexp.MustCompile("^[a-zA-Z0-9-_]*$")
	if !re.MatchString(val) {
		return TaskType(""), fmt.Errorf("task type validation: name %v contains invalid characters", val)
	}

	return TaskType(strings.ToLower(val)), nil
}

// MustNewTaskType instantiates a new TaskType, and panics if a bad input is provided.
func MustNewTaskType(val string) TaskType {
	tt, err := NewTaskType(val)
	if err != nil {
		panic(fmt.Sprintf("%v is not a valid TaskType", val))
	}
	return tt
}

// UnmarshalJSON converts a bytes slice of JSON to a TaskType.
func (t *TaskType) UnmarshalJSON(input []byte) error {
	var aux string
	if err := json.Unmarshal(input, &aux); err != nil {
		return err
	}
	tt, err := NewTaskType(aux)
	*t = tt
	return err
}

// MarshalJSON converts a TaskType to a JSON byte slice.
func (t TaskType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// String returns this TaskType as a string.
func (t TaskType) String() string {
	return string(t)
}

// Value returns this instance serialized for database storage.
func (t TaskType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan reads the database value and returns an instance.
func (t *TaskType) Scan(value interface{}) error {
	temp, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to TaskType", value, value)
	}

	*t = TaskType(temp)
	return nil
}
