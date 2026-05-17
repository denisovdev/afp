package fingerprint

import (
	"sort"
	"strings"
)

// Weights for similarity score computation.
// device_id and hard_fp are handled by the matching cascade (index lookups),
// not by similarity scoring, so they are only used in ComputeFullSimilarity.
type Weights struct {
	// hard (used only for full similarity on device_id/hard_fp match stages)
	DeviceID float64
	HardFP   float64

	// medium — high entropy
	AudioFP             float64
	FontsFP             float64
	WebGLParamsHash     float64
	WebGLExtensionsHash float64
	MediaCodecsHash     float64
	MathFP              float64
	UserAgentFamily     float64
	OSFamily            float64

	// soft — stable numeric/string
	SoftFP         float64
	CPUCores       float64
	DeviceMemoryGB float64
	TimezoneName   float64
	Languages      float64
	ScreenWidth    float64
	ScreenHeight   float64
	PixelRatio     float64
	MaxTouchPoints float64
	ColorDepth     float64
	Platform       float64

	// weak — supplementary
	AvailScreen    float64
	MediaDevices   float64
	ColorGamut     float64
	HDR            float64
	DoNotTrack     float64
	PDFViewer      float64
	ForcedColors   float64
	TimezoneOffset float64
	IntlLocale     float64
}

func DefaultWeights() Weights {
	return Weights{
		DeviceID: 0.45,
		HardFP:   0.25,

		// medium (total 0.62)
		AudioFP:             0.16,
		FontsFP:             0.14,
		WebGLParamsHash:     0.08,
		WebGLExtensionsHash: 0.06,
		MediaCodecsHash:     0.05,
		MathFP:              0.03,
		UserAgentFamily:     0.04,
		OSFamily:            0.06,

		// soft (total 0.27)
		SoftFP:         0.08,
		CPUCores:       0.03,
		DeviceMemoryGB: 0.02,
		TimezoneName:   0.02,
		Languages:      0.03,
		ScreenWidth:    0.01,
		ScreenHeight:   0.01,
		PixelRatio:     0.01,
		MaxTouchPoints: 0.02,
		ColorDepth:     0.01,
		Platform:       0.01,

		// weak signals (total 0.11)
		AvailScreen:    0.02,
		MediaDevices:   0.02,
		ColorGamut:     0.01,
		HDR:            0.005,
		DoNotTrack:     0.005,
		PDFViewer:      0.005,
		ForcedColors:   0.005,
		TimezoneOffset: 0.01,
		IntlLocale:     0.01,
	}
}

// Fields holds all comparable attributes for similarity computation.
type Fields struct {
	// hard
	DeviceID string
	HardFP   string

	// medium
	AudioFP             string
	FontsFP             string
	WebGLParamsHash     string
	WebGLExtensionsHash string
	MediaCodecsHash     string
	MathFP              string
	UserAgentFamily     string
	OSFamily            string

	// soft
	SoftFP         string
	CPUCores       *int
	DeviceMemoryGB *int
	TimezoneName   string
	Languages      []string
	ScreenWidth    *int
	ScreenHeight   *int
	PixelRatio     *float64
	MaxTouchPoints *int
	ColorDepth     *int
	Platform       string

	// weak
	AvailScreenWidth  *int
	AvailScreenHeight *int
	MediaDevicesAudio *int
	MediaDevicesVideo *int
	MediaDevicesOut   *int
	ColorGamut        string
	HDR               *bool
	DoNotTrack        *string
	PDFViewer         *bool
	ForcedColors      *bool
	TimezoneOffset    *int
	IntlLocale        string
}

func eqStr(a, b string) float64 {
	if a != "" && b != "" && strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) {
		return 1.0
	}
	return 0.0
}

func eqIntPtr(a, b *int) float64 {
	if a != nil && b != nil && *a == *b {
		return 1.0
	}
	return 0.0
}

func eqFloatPtr(a, b *float64) float64 {
	if a != nil && b != nil && *a == *b {
		return 1.0
	}
	return 0.0
}

func eqBoolPtr(a, b *bool) float64 {
	if a != nil && b != nil && *a == *b {
		return 1.0
	}
	return 0.0
}

func eqStringPtr(a, b *string) float64 {
	if a != nil && b != nil && strings.EqualFold(strings.TrimSpace(*a), strings.TrimSpace(*b)) {
		return 1.0
	}
	return 0.0
}

func eqLanguages(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	sa := make([]string, len(a))
	copy(sa, a)
	sort.Strings(sa)
	sb := make([]string, len(b))
	copy(sb, b)
	sort.Strings(sb)
	if len(sa) != len(sb) {
		return 0.0
	}
	for i := range sa {
		if !strings.EqualFold(sa[i], sb[i]) {
			return 0.0
		}
	}
	return 1.0
}

func eqAvailScreen(a, b Fields) float64 {
	w := eqIntPtr(a.AvailScreenWidth, b.AvailScreenWidth)
	h := eqIntPtr(a.AvailScreenHeight, b.AvailScreenHeight)
	if w == 1.0 && h == 1.0 {
		return 1.0
	}
	return 0.0
}

func eqMediaDevices(a, b Fields) float64 {
	if a.MediaDevicesAudio == nil || b.MediaDevicesAudio == nil {
		return 0.0
	}
	au := eqIntPtr(a.MediaDevicesAudio, b.MediaDevicesAudio)
	vi := eqIntPtr(a.MediaDevicesVideo, b.MediaDevicesVideo)
	ou := eqIntPtr(a.MediaDevicesOut, b.MediaDevicesOut)
	return (au + vi + ou) / 3.0
}

type weightedPair struct {
	weight float64
	match  float64
}

func scorePairs(pairs []weightedPair) (raw, totalWeight float64) {
	for _, p := range pairs {
		totalWeight += p.weight
		raw += p.weight * p.match
	}
	return raw, totalWeight
}

func softPairs(a, b Fields, w Weights) []weightedPair {
	return []weightedPair{
		// medium
		{w.AudioFP, eqStr(a.AudioFP, b.AudioFP)},
		{w.FontsFP, eqStr(a.FontsFP, b.FontsFP)},
		{w.WebGLParamsHash, eqStr(a.WebGLParamsHash, b.WebGLParamsHash)},
		{w.WebGLExtensionsHash, eqStr(a.WebGLExtensionsHash, b.WebGLExtensionsHash)},
		{w.MediaCodecsHash, eqStr(a.MediaCodecsHash, b.MediaCodecsHash)},
		{w.MathFP, eqStr(a.MathFP, b.MathFP)},
		{w.UserAgentFamily, eqStr(a.UserAgentFamily, b.UserAgentFamily)},
		{w.OSFamily, eqStr(a.OSFamily, b.OSFamily)},

		// soft
		{w.SoftFP, eqStr(a.SoftFP, b.SoftFP)},
		{w.CPUCores, eqIntPtr(a.CPUCores, b.CPUCores)},
		{w.DeviceMemoryGB, eqIntPtr(a.DeviceMemoryGB, b.DeviceMemoryGB)},
		{w.TimezoneName, eqStr(a.TimezoneName, b.TimezoneName)},
		{w.Languages, eqLanguages(a.Languages, b.Languages)},
		{w.ScreenWidth, eqIntPtr(a.ScreenWidth, b.ScreenWidth)},
		{w.ScreenHeight, eqIntPtr(a.ScreenHeight, b.ScreenHeight)},
		{w.PixelRatio, eqFloatPtr(a.PixelRatio, b.PixelRatio)},
		{w.MaxTouchPoints, eqIntPtr(a.MaxTouchPoints, b.MaxTouchPoints)},
		{w.ColorDepth, eqIntPtr(a.ColorDepth, b.ColorDepth)},
		{w.Platform, eqStr(a.Platform, b.Platform)},

		// weak
		{w.AvailScreen, eqAvailScreen(a, b)},
		{w.MediaDevices, eqMediaDevices(a, b)},
		{w.ColorGamut, eqStr(a.ColorGamut, b.ColorGamut)},
		{w.HDR, eqBoolPtr(a.HDR, b.HDR)},
		{w.DoNotTrack, eqStringPtr(a.DoNotTrack, b.DoNotTrack)},
		{w.PDFViewer, eqBoolPtr(a.PDFViewer, b.PDFViewer)},
		{w.ForcedColors, eqBoolPtr(a.ForcedColors, b.ForcedColors)},
		{w.TimezoneOffset, eqIntPtr(a.TimezoneOffset, b.TimezoneOffset)},
		{w.IntlLocale, eqStr(a.IntlLocale, b.IntlLocale)},
	}
}

// ComputeFullSimilarity uses all features including device_id and hard_fp.
func ComputeFullSimilarity(a, b Fields, w Weights) float64 {
	pairs := append([]weightedPair{
		{w.DeviceID, eqStr(a.DeviceID, b.DeviceID)},
		{w.HardFP, eqStr(a.HardFP, b.HardFP)},
	}, softPairs(a, b, w)...)

	raw, totalWeight := scorePairs(pairs)
	if totalWeight == 0 {
		return 0
	}
	return clamp01(raw / totalWeight)
}

// ComputeSoftSimilarity uses only soft/medium/weak features, normalized to
// [0,1] by their total weight. Used at soft_fp and candidate matching stages
// where device_id and hard_fp are already known not to match.
func ComputeSoftSimilarity(a, b Fields, w Weights) float64 {
	pairs := softPairs(a, b, w)
	raw, totalWeight := scorePairs(pairs)
	if totalWeight == 0 {
		return 0
	}
	return clamp01(raw / totalWeight)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
