package domain

import (
	"encoding/json"
	"net/netip"
	"time"
)

type BrowserProfile struct {
	BrowserProfileID int64  `json:"browser_profile_id"`
	PublicID         string `json:"public_id"`
	DeviceClusterID  *int64 `json:"device_cluster_id"`
	HardwareFP       string `json:"hardware_fp"`
	CurrentDeviceID  string `json:"current_device_id"`
	CurrentCanvasFP  string `json:"current_canvas_fp"`
	CurrentWebGLVendor   string `json:"current_webgl_vendor"`
	CurrentWebGLRenderer string `json:"current_webgl_renderer"`
	CurrentHardFP string `json:"current_hard_fp"`
	CurrentSoftFP string `json:"current_soft_fp"`

	// medium-stability signals
	AudioFP             string `json:"audio_fp"`
	FontsFP             string `json:"fonts_fp"`
	WebGLParamsHash     string `json:"webgl_params_hash"`
	WebGLExtensionsHash string `json:"webgl_extensions_hash"`
	MathFP              string `json:"math_fp"`
	MediaCodecsHash     string `json:"media_codecs_hash"`

	UserAgentFamily string `json:"user_agent_family"`
	OSFamily        string `json:"os_family"`

	// soft signals
	CPUCores       *int     `json:"cpu_cores"`
	DeviceMemoryGB *int     `json:"device_memory_gb"`
	TimezoneName   string   `json:"timezone_name"`
	LanguageCode   string   `json:"language_code"`
	Languages      []string `json:"languages"`
	ScreenWidth    *int     `json:"screen_width"`
	ScreenHeight   *int     `json:"screen_height"`
	PixelRatio     *float64 `json:"pixel_ratio"`
	ColorDepth     *int     `json:"color_depth"`
	MaxTouchPoints *int     `json:"max_touch_points"`
	Platform       string   `json:"platform"`

	// weak signals stored as jsonb
	StableAttrs   json.RawMessage `json:"stable_attrs"`
	VariableAttrs json.RawMessage `json:"variable_attrs"`

	FirstSeen         time.Time `json:"first_seen"`
	LastSeen          time.Time `json:"last_seen"`
	EventCount        int       `json:"event_count"`
	LinkedAccountsCnt int       `json:"linked_accounts_cnt"`
}

type FingerprintEvent struct {
	EventID          int64  `json:"event_id"`
	BrowserProfileID *int64 `json:"browser_profile_id"`
	AccountID        string `json:"account_id"`
	ObservedAt       time.Time `json:"observed_at"`
	HardwareFP       string `json:"hardware_fp"`

	// hard
	DeviceID      string `json:"device_id"`
	CanvasFP      string `json:"canvas_fp"`
	WebGLVendor   string `json:"webgl_vendor"`
	WebGLRenderer string `json:"webgl_renderer"`
	HardFP        string `json:"hard_fp"`
	SoftFP        string `json:"soft_fp"`

	// medium
	AudioFP             string `json:"audio_fp"`
	FontsFP             string `json:"fonts_fp"`
	WebGLParamsHash     string `json:"webgl_params_hash"`
	WebGLExtensionsHash string `json:"webgl_extensions_hash"`
	MathFP              string `json:"math_fp"`
	MediaCodecsHash     string `json:"media_codecs_hash"`

	UserAgentRaw    string `json:"user_agent_raw"`
	UserAgentFamily string `json:"user_agent_family"`
	OSFamily        string `json:"os_family"`

	// soft
	CPUCores       *int     `json:"cpu_cores"`
	DeviceMemoryGB *int     `json:"device_memory_gb"`
	TimezoneName   string   `json:"timezone_name"`
	LanguageCode   string   `json:"language_code"`
	Languages      []string `json:"languages"`
	ScreenWidth    *int     `json:"screen_width"`
	ScreenHeight   *int     `json:"screen_height"`
	PixelRatio     *float64 `json:"pixel_ratio"`
	ColorDepth     *int     `json:"color_depth"`
	MaxTouchPoints *int     `json:"max_touch_points"`
	Platform       string   `json:"platform"`

	IPAddr *netip.Addr     `json:"ip_addr"`
	Attrs  json.RawMessage `json:"attrs"`
}

type DeviceCluster struct {
	DeviceClusterID int64     `json:"device_cluster_id"`
	PublicID        string    `json:"public_id"`
	HardwareFP      string    `json:"hardware_fp"`
	WebGLVendor     string    `json:"webgl_vendor"`
	OSFamily        string    `json:"os_family"`
	CPUCores        *int      `json:"cpu_cores"`
	DeviceMemoryGB  *int      `json:"device_memory_gb"`
	ScreenWidth     *int      `json:"screen_width"`
	ScreenHeight    *int      `json:"screen_height"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	ProfilesCount   int       `json:"profiles_count"`
	LinkedAccountsCnt int     `json:"linked_accounts_cnt"`
}

type AccountProfileLink struct {
	AccountID        string    `json:"account_id"`
	BrowserProfileID int64     `json:"browser_profile_id"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	EventsCount      int       `json:"events_count"`
}

const (
	MatchTypeDeviceID  = "device_id"
	MatchTypeHardFP    = "hard_fp"
	MatchTypeSoftFP    = "soft_fp"
	MatchTypeCandidate = "candidate"
	MatchTypeNew       = "new_profile"
)
