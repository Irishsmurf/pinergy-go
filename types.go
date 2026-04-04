package pinergy

import (
	"strconv"
	"time"
)

// UnixTime is a time.Time that unmarshals from a JSON string containing a
// Unix timestamp (e.g. "1773446400"). The Pinergy API returns all timestamps
// in this format rather than RFC 3339.
type UnixTime struct {
	time.Time
}

// zeroUnixTimeBytes is returned when UnixTime is zero.
// Warning: The returned slice is mutable. Do not modify it!
var zeroUnixTimeBytes = []byte(`"0"`)

// UnmarshalJSON implements json.Unmarshaler.
func (u *UnixTime) UnmarshalJSON(b []byte) error {
	// Fast path: strip quotes directly without allocating strings.
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}

	if len(b) == 0 || string(b) == "null" {
		u.Time = time.Time{}
		return nil
	}

	n, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}
	u.Time = time.Unix(n, 0).UTC()
	return nil
}

// MarshalJSON implements json.Marshaler.
func (u UnixTime) MarshalJSON() ([]byte, error) {
	if u.IsZero() {
		return zeroUnixTimeBytes, nil
	}
	b := make([]byte, 0, 22)
	b = append(b, '"')
	b = strconv.AppendInt(b, u.Unix(), 10)
	b = append(b, '"')
	return b, nil
}

// ---------------------------------------------------------------------------
// Auth types
// ---------------------------------------------------------------------------

// LoginRequest is sent to POST /api/login/.
type LoginRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`     // SHA-1 hex of plaintext password
	DeviceToken string `json:"device_token"` // FCM token; empty string for headless use
}

// User contains the authenticated user's profile returned by Login.
type User struct {
	Title              string `json:"title"`
	Name               string `json:"name"`
	PineryID           string `json:"pinergy_id"`
	MobileNumber       string `json:"mobile_number"`
	SMSNotifications   bool   `json:"sms_notifications"`
	EmailNotifications bool   `json:"email_notifications"`
	FirstName          string `json:"firstName"`
	LastName           string `json:"lastName"`
}

// House contains the property profile returned by Login.
type House struct {
	Type          int `json:"type"`
	HeatingType   int `json:"heating_type"`
	BedroomCount  int `json:"bedroom_count"`
	AdultCount    int `json:"adult_count"`
	ChildrenCount int `json:"children_count"`
}

// CreditCard is a saved payment method returned by Login.
type CreditCard struct {
	Token string `json:"cc_token"`
	Name  string `json:"name"`
	Last4 string `json:"last_4_digits"`
}

// LoginResponse is the response from POST /api/login/.
type LoginResponse struct {
	Success        bool         `json:"success"`
	ErrorCode      int          `json:"error_code"`
	Message        *string      `json:"message"`
	AuthToken      string       `json:"auth_token"`
	IsLegacyMeter  bool         `json:"is_legacy_meter"`
	IsNoWANMeter   bool         `json:"is_no_wan_meter"`
	IsLevelPay     bool         `json:"is_level_pay"`
	IsChild        bool         `json:"is_child"`
	IsBusinessConn bool         `json:"is_business_connect"`
	PremisesNumber string       `json:"premises_number"`
	AccountType    string       `json:"account_type"`
	User           User         `json:"user"`
	House          House        `json:"house"`
	CreditCards    []CreditCard `json:"credit_cards"`
}

// ---------------------------------------------------------------------------
// Balance types
// ---------------------------------------------------------------------------

// BalanceResponse is the response from GET /api/balance/.
type BalanceResponse struct {
	Success         bool     `json:"success"`
	Balance         float64  `json:"balance"`
	TopUpInDays     int      `json:"top_up_in_days"`
	PendingTopUp    bool     `json:"pending_top_up"`
	PendingTopUpBy  string   `json:"pending_top_up_by"`
	LastTopUpTime   UnixTime `json:"last_top_up_time"`
	LastTopUpAmount float64  `json:"last_top_up_amount"`
	CreditLow       bool     `json:"credit_low"`
	EmergencyCredit bool     `json:"emergency_credit"`
	PowerOff        bool     `json:"power_off"`
	LastReading     UnixTime `json:"last_reading"`
}

// ---------------------------------------------------------------------------
// Usage types
// ---------------------------------------------------------------------------

// UsageEntry is one data point in a usage period (day, week, or month).
type UsageEntry struct {
	Available bool     `json:"available"`
	Amount    float64  `json:"amount"` // cost in EUR
	KWh       float64  `json:"kwh"`
	CO2       float64  `json:"co2"`
	Date      UnixTime `json:"date"`
}

// UsageResponse is the response from GET /api/usage/.
type UsageResponse struct {
	Success bool         `json:"success"`
	Day     []UsageEntry `json:"day"`
	Week    []UsageEntry `json:"week"`
	Month   []UsageEntry `json:"month"`
}

// LevelPayDailyValue holds kWh per tariff band for a single interval.
type LevelPayDailyValue struct {
	Label  string             `json:"label"`
	DayKWh map[string]float64 `json:"daykWh"`
}

// LevelPayDaily holds the half-hourly time-of-use data structure.
type LevelPayDaily struct {
	Labels []string             `json:"labels"`
	Flags  []string             `json:"flags"`
	Values []LevelPayDailyValue `json:"values"`
}

// LevelPayUsageData wraps the daily interval data.
type LevelPayUsageData struct {
	Daily LevelPayDaily `json:"daily"`
}

// LevelPayUsageResponse is the response from GET /api/levelpayusage/.
type LevelPayUsageResponse struct {
	UsageData LevelPayUsageData `json:"usageData"`
}

// ---------------------------------------------------------------------------
// Comparison types
// ---------------------------------------------------------------------------

// CompareMetric holds the user's and the average home's value for one metric.
type CompareMetric struct {
	UsersHome   float64 `json:"users_home"`
	AverageHome float64 `json:"average_home"`
}

// ComparePeriod holds comparison data for one time period.
type ComparePeriod struct {
	Available bool          `json:"available"`
	Euro      CompareMetric `json:"euro"`
	KWh       CompareMetric `json:"kwh"`
	CO2       CompareMetric `json:"co2"`
}

// CompareResponse is the response from GET /api/compare/.
type CompareResponse struct {
	Success bool          `json:"success"`
	Day     ComparePeriod `json:"day"`
	Week    ComparePeriod `json:"week"`
	Month   ComparePeriod `json:"month"`
}

// ---------------------------------------------------------------------------
// Top-up types
// ---------------------------------------------------------------------------

// ScheduledTopUp is a top-up configured to run on a fixed day of the month.
type ScheduledTopUp struct {
	CurrentUser bool    `json:"current_user"`
	TopUpAmount float64 `json:"top_up_amount"`
	TopUpDay    int     `json:"top_up_day"`
	Customer    string  `json:"customer"`
}

// ActiveTopUpsResponse is the response from GET /api/activetopups/.
type ActiveTopUpsResponse struct {
	Success    bool             `json:"success"`
	Scheduled  []ScheduledTopUp `json:"scheduled"`
	AutoTopUps []any            `json:"auto_top_ups"`
}

// UpdateDeviceTokenRequest is the body sent to POST /api/updatedevicetoken/.
type UpdateDeviceTokenRequest struct {
	DeviceToken string `json:"device_token"`
	DeviceType  string `json:"device_type"`
	OSVersion   string `json:"os_version"`
}

// ---------------------------------------------------------------------------
// Config types
// ---------------------------------------------------------------------------

// ConfigInfoResponse is the response from GET /api/configinfo/.
type ConfigInfoResponse struct {
	Success               bool      `json:"success"`
	Thresholds            []float64 `json:"thresholds"`
	TopUpAmounts          []float64 `json:"top_up_amounts"`
	AutoUpAmounts         []float64 `json:"auto_up_amounts"`
	ScheduledTopUpAmounts []float64 `json:"scheduled_top_up_amounts"`
}

// NamedItem is a { id, name } pair used in defaults info.
type NamedItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// DefaultsInfoResponse is the response from GET /api/defaultsinfo/.
type DefaultsInfoResponse struct {
	Success         bool        `json:"success"`
	HouseTypes      []NamedItem `json:"house_types"`
	HeatingTypes    []NamedItem `json:"heating_types"`
	MaxBedrooms     int         `json:"max_bedrooms"`
	DefaultBedrooms int         `json:"default_bedrooms"`
	MaxAdults       int         `json:"max_adults"`
	DefaultAdults   int         `json:"default_adults"`
	MaxChildren     int         `json:"max_children"`
	DefaultChildren int         `json:"default_children"`
}

// ---------------------------------------------------------------------------
// Notification types
// ---------------------------------------------------------------------------

// NotificationResponse is the response from GET /api/getnotif/.
type NotificationResponse struct {
	Success           bool   `json:"success"`
	SMS               bool   `json:"sms"`
	Email             bool   `json:"email"`
	Phone             bool   `json:"phone"`
	ShouldShow        int    `json:"should_show"`
	ShouldShowMessage string `json:"should_show_message"`
}

// ---------------------------------------------------------------------------
// Version types
// ---------------------------------------------------------------------------

// VersionResponse is the response from GET /version.json.
// The exact schema is not fully documented; fields are best-effort.
type VersionResponse struct {
	MinVersion     string `json:"min_version"`
	CurrentVersion string `json:"current_version"`
	ForceUpdate    bool   `json:"force_update"`
}

// ---------------------------------------------------------------------------
// Generic envelope
// ---------------------------------------------------------------------------

// envelope is used internally to check the success field before decoding
// the full response type.
type envelope struct {
	Success   bool   `json:"success"`
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}
