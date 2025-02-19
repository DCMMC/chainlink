package models

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm/schema"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/DCMMC/chainlink/core/assets"
	"github.com/DCMMC/chainlink/core/utils"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// CronParser is the global parser for crontabs.
// It accepts the standard 5 field cron syntax as well as an optional 6th field
// at the front to represent seconds.
var CronParser cron.Parser

func init() {
	cronParserSpec := cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor
	CronParser = cron.NewParser(cronParserSpec)
}

// JSON stores the json types string, number, bool, and null.
// Arrays and Objects are returned as their raw json types.
type JSON struct {
	gjson.Result
}

func (JSON) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (JSON) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "JSONB"
	}
	return ""
}

// Value returns this instance serialized for database storage.
func (j JSON) Value() (driver.Value, error) {
	s := j.Bytes()
	if len(s) == 0 {
		return nil, nil
	}
	return s, nil
}

// Scan reads the database value and returns an instance.
func (j *JSON) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*j = JSON{Result: gjson.Parse(v)}
	case []byte:
		*j = JSON{Result: gjson.ParseBytes(v)}
	default:
		return fmt.Errorf("unable to convert %v of %T to JSON", value, value)
	}
	return nil
}

func MustParseJSON(b []byte) JSON {
	var j JSON
	str := string(b)
	if len(str) == 0 {
		panic("empty byte array")
	}
	if err := json.Unmarshal([]byte(str), &j); err != nil {
		panic(err)
	}
	return j
}

// ParseJSON attempts to coerce the input byte array into valid JSON
// and parse it into a JSON object.
func ParseJSON(b []byte) (JSON, error) {
	var j JSON
	str := string(b)
	if len(str) == 0 {
		return j, nil
	}
	return j, json.Unmarshal([]byte(str), &j)
}

// UnmarshalJSON parses the JSON bytes and stores in the *JSON pointer.
func (j *JSON) UnmarshalJSON(b []byte) error {
	str := string(b)
	if !gjson.Valid(str) {
		return fmt.Errorf("invalid JSON: %v", str)
	}
	*j = JSON{gjson.Parse(str)}
	return nil
}

// MarshalJSON returns the JSON data if it already exists, returns
// an empty JSON object as bytes if not.
func (j JSON) MarshalJSON() ([]byte, error) {
	if j.Exists() {
		return j.Bytes(), nil
	}
	return []byte("{}"), nil
}

func (j *JSON) UnmarshalTOML(val interface{}) error {
	var bs []byte
	switch v := val.(type) {
	case string:
		bs = []byte(v)
	case []byte:
		bs = v
	}
	var err error
	*j, err = ParseJSON(bs)
	return err
}

// Bytes returns the raw JSON.
func (j JSON) Bytes() []byte {
	if len(j.String()) == 0 {
		return nil
	}
	return []byte(j.String())
}

// AsMap returns j as a map
func (j JSON) AsMap() (map[string]interface{}, error) {
	output := make(map[string]interface{})
	switch v := j.Result.Value().(type) {
	case map[string]interface{}:
		for key, value := range v {
			output[key] = value
		}
	case nil:
	default:
		return nil, errors.New("can only add to JSON objects or null")
	}
	return output, nil
}

// mapToJSON returns m as a JSON object, or errors
func mapToJSON(m map[string]interface{}) (JSON, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return JSON{}, err
	}
	return JSON{Result: gjson.ParseBytes(bytes)}, nil
}

// Add returns a new instance of JSON with the new value added.
func (j JSON) Add(insertKey string, insertValue interface{}) (JSON, error) {
	return j.MultiAdd(KV{insertKey: insertValue})
}

func (j JSON) PrependAtArrayKey(insertKey string, insertValue interface{}) (JSON, error) {
	curr := j.Get(insertKey).Array()
	updated := make([]interface{}, 0)
	updated = append(updated, insertValue)
	for _, c := range curr {
		updated = append(updated, c.Value())
	}
	return j.Add(insertKey, updated)
}

// KV represents a key/value pair to be added to a JSON object
type KV map[string]interface{}

// MultiAdd returns a new instance of j with the new values added.
func (j JSON) MultiAdd(keyValues KV) (JSON, error) {
	output, err := j.AsMap()
	if err != nil {
		return JSON{}, err
	}
	for key, value := range keyValues {
		output[key] = value
	}
	return mapToJSON(output)
}

// Delete returns a new instance of JSON with the specified key removed.
func (j JSON) Delete(key string) (JSON, error) {
	js, err := sjson.Delete(j.String(), key)
	if err != nil {
		return j, err
	}
	return ParseJSON([]byte(js))
}

// CBOR returns a bytes array of the JSON map or array encoded to CBOR.
func (j JSON) CBOR() ([]byte, error) {
	switch v := j.Result.Value().(type) {
	case map[string]interface{}, []interface{}, nil:
		return cbor.Marshal(v)
	default:
		var b []byte
		return b, fmt.Errorf("unable to coerce JSON to CBOR for type %T", v)
	}
}

// WebURL contains the URL of the endpoint.
type WebURL url.URL

// UnmarshalJSON parses the raw URL stored in JSON-encoded
// data to a URL structure and sets it to the URL field.
func (w *WebURL) UnmarshalJSON(j []byte) error {
	var v string
	err := json.Unmarshal(j, &v)
	if err != nil {
		return err
	}
	// handle no url case
	if len(v) == 0 {
		return nil
	}

	u, err := url.ParseRequestURI(v)
	if err != nil {
		return err
	}
	*w = WebURL(*u)
	return nil
}

// MarshalJSON returns the JSON-encoded string of the given data.
func (w WebURL) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.String())
}

// String delegates to the wrapped URL struct or an empty string when it is nil
func (w WebURL) String() string {
	url := url.URL(w)
	return url.String()
}

// Value returns this instance serialized for database storage.
func (w WebURL) Value() (driver.Value, error) {
	return w.String(), nil
}

// Scan reads the database value and returns an instance.
func (w *WebURL) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to WebURL", value, value)
	}

	u, err := url.ParseRequestURI(s)
	if err != nil {
		return err
	}
	*w = WebURL(*u)
	return nil
}

// Cron holds the string that will represent the spec of the cron-job.
type Cron string

// UnmarshalJSON parses the raw spec stored in JSON-encoded
// data and stores it to the Cron string.
func (c *Cron) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return fmt.Errorf("Cron: %w", err)
	}
	if s == "" {
		return nil
	}

	if !strings.HasPrefix(s, "CRON_TZ=") {
		return errors.New("Cron: specs must specify a time zone using CRON_TZ, e.g. 'CRON_TZ=UTC 5 * * * *'")
	}

	_, err = CronParser.Parse(s)
	if err != nil {
		return fmt.Errorf("Cron: %w", err)
	}
	*c = Cron(s)
	return nil
}

// String returns the current Cron spec string.
func (c Cron) String() string {
	return string(c)
}

// Duration is a non-negative time duration.
type Duration struct{ d time.Duration }

func MakeDuration(d time.Duration) (Duration, error) {
	if d < time.Duration(0) {
		return Duration{}, fmt.Errorf("cannot make negative time duration: %s", d)
	}
	return Duration{d: d}, nil
}

func MustMakeDuration(d time.Duration) Duration {
	rv, err := MakeDuration(d)
	if err != nil {
		panic(err)
	}
	return rv
}

// Duration returns the value as the standard time.Duration value.
func (d Duration) Duration() time.Duration {
	return d.d
}

// Before returns the time d units before time t
func (d Duration) Before(t time.Time) time.Time {
	return t.Add(-d.Duration())
}

// Shorter returns true if and only if d is shorter than od.
func (d Duration) Shorter(od Duration) bool { return d.d < od.d }

// IsInstant is true if and only if d is of duration 0
func (d Duration) IsInstant() bool { return d.d == 0 }

// String returns a string representing the duration in the form "72h3m0.5s".
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. The zero duration formats as 0s.
func (d Duration) String() string {
	return d.Duration().String()
}

// MarshalJSON implements the json.Marshaler interface.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(input []byte) error {
	var txt string
	err := json.Unmarshal(input, &txt)
	if err != nil {
		return err
	}
	v, err := time.ParseDuration(string(txt))
	if err != nil {
		return err
	}
	*d, err = MakeDuration(v)
	if err != nil {
		return err
	}
	return nil
}

func (d *Duration) Scan(v interface{}) (err error) {
	switch tv := v.(type) {
	case int64:
		*d, err = MakeDuration(time.Duration(tv))
		return err
	default:
		return errors.Errorf(`don't know how to parse "%s" of type %T as a `+
			`models.Duration`, tv, tv)
	}
}

func (d Duration) Value() (driver.Value, error) {
	return int64(d.d), nil
}

// Interval represents a time.Duration stored as a Postgres interval type
type Interval time.Duration

func (i Interval) Duration() time.Duration {
	return time.Duration(i)
}

// MarshalText implements the text.Marshaler interface.
func (i Interval) MarshalText() ([]byte, error) {
	return []byte(time.Duration(i).String()), nil
}

// UnmarshalText implements the text.Unmarshaler interface.
func (i *Interval) UnmarshalText(input []byte) error {
	v, err := time.ParseDuration(string(input))
	if err != nil {
		return err
	}
	*i = Interval(v)
	return nil
}

func (i *Interval) Scan(v interface{}) error {
	if v == nil {
		*i = Interval(time.Duration(0))
		return nil
	}
	asInt64, is := v.(int64)
	if !is {
		return errors.Errorf("models.Interval#Scan() wanted int64, got %T", v)
	}
	*i = Interval(time.Duration(asInt64) * time.Nanosecond)
	return nil
}

func (i Interval) Value() (driver.Value, error) {
	return time.Duration(i).Nanoseconds(), nil
}

func (i Interval) IsZero() bool {
	return time.Duration(i) == time.Duration(0)
}

// SendEtherRequest represents a request to transfer ETH.
type SendEtherRequest struct {
	DestinationAddress common.Address `json:"address"`
	FromAddress        common.Address `json:"from"`
	Amount             assets.Eth     `json:"amount"`
	EVMChainID         *utils.Big     `json:"evmChainID"`
}

// AddressCollection is an array of common.Address
// serializable to and from a database.
type AddressCollection []common.Address

// ToStrings returns this address collection as an array of strings.
func (r AddressCollection) ToStrings() []string {
	// Unable to convert copy-free without unsafe:
	// https://stackoverflow.com/a/48554123/639773
	converted := make([]string, len(r))
	for i, e := range r {
		converted[i] = e.Hex()
	}
	return converted
}

// Value returns the string value to be written to the database.
func (r AddressCollection) Value() (driver.Value, error) {
	return strings.Join(r.ToStrings(), ","), nil
}

// Scan parses the database value as a string.
func (r *AddressCollection) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to AddressCollection", value, value)
	}

	if len(str) == 0 {
		return nil
	}

	arr := strings.Split(str, ",")
	collection := make(AddressCollection, len(arr))
	for i, a := range arr {
		collection[i] = common.HexToAddress(a)
	}
	*r = collection
	return nil
}

// Configuration stores key value pairs for overriding global configuration
type Configuration struct {
	ID        int64  `gorm:"primary_key"`
	Name      string `gorm:"not null;unique;index"`
	Value     string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *gorm.DeletedAt
}

// Merge returns a new map with all keys merged from left to right
// On conflicting keys, rightmost inputs will clobber leftmost inputs
func Merge(inputs ...JSON) (JSON, error) {
	output := make(map[string]interface{})

	for _, input := range inputs {
		switch v := input.Result.Value().(type) {
		case map[string]interface{}:
			for key, value := range v {
				output[key] = value
			}
		case nil:
		default:
			return JSON{}, errors.New("can only merge JSON objects")
		}
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return JSON{}, err
	}

	return JSON{Result: gjson.ParseBytes(bytes)}, nil
}

// Explicit type indicating a 32-byte sha256 hash
type Sha256Hash [32]byte

// MarshalJSON converts a Sha256Hash to a JSON byte slice.
func (s Sha256Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON converts a bytes slice of JSON to a TaskType.
func (s *Sha256Hash) UnmarshalJSON(input []byte) error {
	var shaHash string
	if err := json.Unmarshal(input, &shaHash); err != nil {
		return err
	}

	sha, err := Sha256HashFromHex(shaHash)
	if err != nil {
		return err
	}

	*s = sha
	return nil
}

func Sha256HashFromHex(x string) (Sha256Hash, error) {
	bs, err := hex.DecodeString(x)
	if err != nil {
		return Sha256Hash{}, err
	}
	var hash Sha256Hash
	copy(hash[:], bs)
	return hash, nil
}

func MustSha256HashFromHex(x string) Sha256Hash {
	bs, err := hex.DecodeString(x)
	if err != nil {
		panic(err)
	}
	var hash Sha256Hash
	copy(hash[:], bs)
	return hash
}

func (s Sha256Hash) String() string {
	return hex.EncodeToString(s[:])
}

func (s *Sha256Hash) UnmarshalText(bs []byte) error {
	x, err := hex.DecodeString(string(bs))
	if err != nil {
		return err
	}
	*s = Sha256Hash{}
	copy((*s)[:], x)
	return nil
}

func (s *Sha256Hash) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.Errorf("Failed to unmarshal Sha256Hash value: %v", value)
	}
	if s == nil {
		*s = Sha256Hash{}
	}
	copy((*s)[:], bytes)
	return nil
}

func (s Sha256Hash) Value() (driver.Value, error) {
	b := make([]byte, 32)
	copy(b, s[:])
	return b, nil
}
