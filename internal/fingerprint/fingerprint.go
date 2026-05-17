package fingerprint

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func intPtrStr(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

func floatPtrStr(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *v)
}

func hashFields(fields ...string) string {
	h := sha256.New()
	for i, f := range fields {
		if i > 0 {
			h.Write([]byte("|"))
		}
		h.Write([]byte(normalize(f)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func normalizeLanguages(langs []string) string {
	if len(langs) == 0 {
		return ""
	}
	sorted := make([]string, len(langs))
	copy(sorted, langs)
	sort.Strings(sorted)
	for i := range sorted {
		sorted[i] = normalize(sorted[i])
	}
	return strings.Join(sorted, ",")
}

// ComputeHardFP builds a SHA-256 hash from stable browser attributes.
func ComputeHardFP(deviceID, canvasFP, webglVendor, webglRenderer, osFamily, uaFamily string) string {
	return hashFields(deviceID, canvasFP, webglVendor, webglRenderer, osFamily, uaFamily)
}

// ComputeHardwareFP builds a SHA-256 hash from browser-independent hardware
// attributes. This hash is the same across different browsers on the same device.
//
// Excluded from the hash (browser-dependent despite appearing hardware-related):
//   - webgl_vendor: Chrome/ANGLE reports differently from Firefox (native OpenGL)
//   - webgl_params_hash: WebGL parameter values differ across browser engines
//   - device_memory_gb: not supported in Firefox (always null)
func ComputeHardwareFP(
	cpuCores *int,
	screenWidth, screenHeight *int,
	pixelRatio *float64,
	maxTouchPoints, colorDepth *int,
	osFamily, platform, timezoneName string,
) string {
	return hashFields(
		intPtrStr(cpuCores),
		intPtrStr(screenWidth),
		intPtrStr(screenHeight),
		floatPtrStr(pixelRatio),
		intPtrStr(maxTouchPoints),
		intPtrStr(colorDepth),
		osFamily,
		platform,
		timezoneName,
	)
}

// ComputeSoftFP builds a SHA-256 hash from medium-stability and soft attributes.
// Includes enough medium signals to significantly reduce collision probability.
func ComputeSoftFP(
	osFamily string,
	cpuCores, deviceMemoryGB *int,
	timezoneName, languageCode string,
	screenWidth, screenHeight *int,
	pixelRatio *float64,
	audioFP, fontsFP, webglParamsHash, webglExtensionsHash string,
	mathFP, mediaCodecsHash string,
	maxTouchPoints, colorDepth *int,
	platform string,
	languages []string,
) string {
	return hashFields(
		osFamily,
		intPtrStr(cpuCores),
		intPtrStr(deviceMemoryGB),
		timezoneName,
		languageCode,
		intPtrStr(screenWidth),
		intPtrStr(screenHeight),
		floatPtrStr(pixelRatio),
		audioFP,
		fontsFP,
		webglParamsHash,
		webglExtensionsHash,
		mathFP,
		mediaCodecsHash,
		intPtrStr(maxTouchPoints),
		intPtrStr(colorDepth),
		platform,
		normalizeLanguages(languages),
	)
}
