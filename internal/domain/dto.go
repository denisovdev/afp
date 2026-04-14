package domain

import (
	"encoding/json"
	"time"
)

type FingerprintInput struct {
	AccountID  int64      `json:"account_id"`
	ObservedAt *time.Time `json:"observed_at"`

	// hard
	DeviceID      string `json:"device_id"`
	CanvasFP      string `json:"canvas_fp"`
	WebGLVendor   string `json:"webgl_vendor"`
	WebGLRenderer string `json:"webgl_renderer"`

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

	// weak — stored as-is in attrs jsonb
	AvailScreenWidth  *int    `json:"avail_screen_width"`
	AvailScreenHeight *int    `json:"avail_screen_height"`
	DoNotTrack        *string `json:"do_not_track"`
	PDFViewerEnabled  *bool   `json:"pdf_viewer_enabled"`
	ColorGamut        string  `json:"color_gamut"`
	HDR               *bool   `json:"hdr"`
	ForcedColors      *bool   `json:"forced_colors"`
	PrefersColorScheme string `json:"prefers_color_scheme"`
	MediaDevicesCount  *MediaDevicesCount `json:"media_devices_count"`
	TimezoneOffset    *int    `json:"timezone_offset"`
	IntlLocale        string  `json:"intl_locale"`

	IPAddr string          `json:"ip_addr"`
	Attrs  json.RawMessage `json:"attrs"`
}

type MediaDevicesCount struct {
	AudioInput  int `json:"audioinput"`
	VideoInput  int `json:"videoinput"`
	AudioOutput int `json:"audiooutput"`
}

func (f *FingerprintInput) GetObservedAt() time.Time {
	if f.ObservedAt != nil {
		return *f.ObservedAt
	}
	return time.Now()
}

type ResolveResult struct {
	BrowserProfileID          int64   `json:"browser_profile_id"`
	DeviceClusterID           int64   `json:"device_cluster_id"`
	MatchType                 string  `json:"match_type"`
	Score                     float64 `json:"score"`
	IsMultiAccountSuspected   bool    `json:"is_multi_account_suspected"`
	LinkedAccountsCount       int     `json:"linked_accounts_count"`
	DeviceLinkedAccountsCount int     `json:"device_linked_accounts_count"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
