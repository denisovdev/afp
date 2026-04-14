package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/netip"
	"time"

	"github.com/dxngee/antifraud-processing/internal/domain"
	"github.com/dxngee/antifraud-processing/internal/fingerprint"
	"github.com/dxngee/antifraud-processing/internal/repository"
)

const candidateLimit = 50

type FingerprintService struct {
	profiles  repository.ProfileRepository
	events    repository.EventRepository
	links     repository.LinkRepository
	clusters  repository.ClusterRepository
	txm       repository.TxManager
	threshold float64
	weights   fingerprint.Weights
}

func NewFingerprintService(
	profiles repository.ProfileRepository,
	events repository.EventRepository,
	links repository.LinkRepository,
	clusters repository.ClusterRepository,
	txm repository.TxManager,
	threshold float64,
) *FingerprintService {
	return &FingerprintService{
		profiles:  profiles,
		events:    events,
		links:     links,
		clusters:  clusters,
		txm:       txm,
		threshold: threshold,
		weights:   fingerprint.DefaultWeights(),
	}
}

func (s *FingerprintService) ResolveOrCreateProfile(ctx context.Context, input domain.FingerprintInput) (domain.ResolveResult, error) {
	observedAt := input.GetObservedAt()

	hardFP := fingerprint.ComputeHardFP(
		input.DeviceID, input.CanvasFP,
		input.WebGLVendor, input.WebGLRenderer,
		input.OSFamily, input.UserAgentFamily,
	)
	softFP := fingerprint.ComputeSoftFP(
		input.OSFamily, input.CPUCores, input.DeviceMemoryGB,
		input.TimezoneName, input.LanguageCode,
		input.ScreenWidth, input.ScreenHeight, input.PixelRatio,
		input.AudioFP, input.FontsFP,
		input.WebGLParamsHash, input.WebGLExtensionsHash,
		input.MathFP, input.MediaCodecsHash,
		input.MaxTouchPoints, input.ColorDepth,
		input.Platform, input.Languages,
	)
	hardwareFP := fingerprint.ComputeHardwareFP(
		input.CPUCores,
		input.ScreenWidth, input.ScreenHeight, input.PixelRatio,
		input.MaxTouchPoints, input.ColorDepth,
		input.OSFamily, input.Platform, input.TimezoneName,
	)

	var result domain.ResolveResult

	err := s.txm.WithTx(ctx, func(txCtx context.Context) error {
		matchedProfile, matchType, score, err := s.findMatch(txCtx, input, hardFP, softFP)
		if err != nil {
			return err
		}

		// --- device cluster ---
		// Resolve the cluster for the CURRENT request's hardware fingerprint.
		requestClusterID, _, err := s.resolveCluster(txCtx, hardwareFP, input, observedAt)
		if err != nil {
			return err
		}

		var profileID int64
		// profileClusterID is the cluster the browser_profile actually belongs to.
		// For existing profiles that already have a cluster, we keep it unchanged —
		// a candidate match from a different device must not reassign the profile.
		var profileClusterID int64

		if matchedProfile != nil {
			profileID = matchedProfile.BrowserProfileID

			if matchedProfile.DeviceClusterID != nil {
				profileClusterID = *matchedProfile.DeviceClusterID
			} else {
				profileClusterID = requestClusterID
			}

			s.updateProfile(matchedProfile, input, hardFP, softFP, hardwareFP, profileClusterID, observedAt)
			if err := s.profiles.UpdateBrowserProfile(txCtx, matchedProfile); err != nil {
				return err
			}
		} else {
			matchType = domain.MatchTypeNew
			score = 0
			profileClusterID = requestClusterID
			profile := s.buildNewProfile(input, hardFP, softFP, hardwareFP, profileClusterID, observedAt)
			profileID, err = s.profiles.InsertBrowserProfile(txCtx, profile)
			if err != nil {
				return err
			}
		}

		event := s.buildEvent(input, profileID, hardFP, softFP, hardwareFP, observedAt)
		if _, err := s.events.InsertFingerprintEvent(txCtx, event); err != nil {
			return err
		}

		if err := s.links.UpsertAccountProfileLink(txCtx, input.AccountID, profileID); err != nil {
			return err
		}

		linkedCount, err := s.links.CountLinkedAccounts(txCtx, profileID)
		if err != nil {
			return err
		}

		if matchedProfile != nil {
			matchedProfile.LinkedAccountsCnt = linkedCount
			if err := s.profiles.UpdateBrowserProfile(txCtx, matchedProfile); err != nil {
				return err
			}
		}

		// Refresh counters for the profile's cluster (not the request's cluster).
		deviceLinkedAccounts, err := s.clusters.CountClusterLinkedAccounts(txCtx, profileClusterID)
		if err != nil {
			return err
		}
		profilesCount, err := s.clusters.CountClusterProfiles(txCtx, profileClusterID)
		if err != nil {
			return err
		}

		profileCluster, err := s.clusters.FindClusterByHardwareFP(txCtx,
			func() string {
				if matchedProfile != nil && matchedProfile.HardwareFP != "" {
					return matchedProfile.HardwareFP
				}
				return hardwareFP
			}(),
		)
		if err != nil {
			return err
		}
		if profileCluster != nil {
			profileCluster.LinkedAccountsCnt = deviceLinkedAccounts
			profileCluster.ProfilesCount = profilesCount
			profileCluster.LastSeen = observedAt
			if err := s.clusters.UpdateDeviceCluster(txCtx, profileCluster); err != nil {
				return err
			}
		}

		// Also update the request's cluster if it's different from the profile's.
		if requestClusterID != profileClusterID {
			reqDeviceAccounts, err := s.clusters.CountClusterLinkedAccounts(txCtx, requestClusterID)
			if err != nil {
				return err
			}
			reqProfiles, err := s.clusters.CountClusterProfiles(txCtx, requestClusterID)
			if err != nil {
				return err
			}
			reqCluster, err := s.clusters.FindClusterByHardwareFP(txCtx, hardwareFP)
			if err != nil {
				return err
			}
			if reqCluster != nil {
				reqCluster.LinkedAccountsCnt = reqDeviceAccounts
				reqCluster.ProfilesCount = reqProfiles
				reqCluster.LastSeen = observedAt
				if err := s.clusters.UpdateDeviceCluster(txCtx, reqCluster); err != nil {
					return err
				}
			}
		}

		result = domain.ResolveResult{
			BrowserProfileID:          profileID,
			DeviceClusterID:           profileClusterID,
			MatchType:                 matchType,
			Score:                     score,
			IsMultiAccountSuspected:   linkedCount > 1 || deviceLinkedAccounts > 1,
			LinkedAccountsCount:       linkedCount,
			DeviceLinkedAccountsCount: deviceLinkedAccounts,
		}

		return nil
	})

	if err != nil {
		return domain.ResolveResult{}, err
	}

	slog.Info("fingerprint resolved",
		"browser_profile_id", result.BrowserProfileID,
		"device_cluster_id", result.DeviceClusterID,
		"match_type", result.MatchType,
		"score", result.Score,
		"linked_accounts", result.LinkedAccountsCount,
		"device_linked_accounts", result.DeviceLinkedAccountsCount,
	)

	return result, nil
}

// resolveCluster finds an existing device cluster by hardware_fp or creates one.
func (s *FingerprintService) resolveCluster(ctx context.Context, hardwareFP string, input domain.FingerprintInput, observedAt time.Time) (int64, int, error) {
	cluster, err := s.clusters.FindClusterByHardwareFP(ctx, hardwareFP)
	if err != nil {
		return 0, 0, err
	}

	if cluster != nil {
		return cluster.DeviceClusterID, cluster.LinkedAccountsCnt, nil
	}

	newCluster := &domain.DeviceCluster{
		HardwareFP:     hardwareFP,
		WebGLVendor:    input.WebGLVendor,
		OSFamily:       input.OSFamily,
		CPUCores:       input.CPUCores,
		DeviceMemoryGB: input.DeviceMemoryGB,
		ScreenWidth:    input.ScreenWidth,
		ScreenHeight:   input.ScreenHeight,
		FirstSeen:      observedAt,
		LastSeen:       observedAt,
		ProfilesCount:  1,
	}
	clusterID, err := s.clusters.InsertDeviceCluster(ctx, newCluster)
	if err != nil {
		return 0, 0, err
	}
	return clusterID, 0, nil
}

func (s *FingerprintService) findMatch(ctx context.Context, input domain.FingerprintInput, hardFP, softFP string) (*domain.BrowserProfile, string, float64, error) {
	if input.DeviceID != "" {
		p, err := s.profiles.FindProfileByDeviceID(ctx, input.DeviceID)
		if err != nil {
			return nil, "", 0, err
		}
		if p != nil {
			score := s.computeFullScore(input, p, hardFP, softFP)
			return p, domain.MatchTypeDeviceID, score, nil
		}
	}

	p, err := s.profiles.FindProfileByHardFP(ctx, hardFP)
	if err != nil {
		return nil, "", 0, err
	}
	if p != nil {
		score := s.computeFullScore(input, p, hardFP, softFP)
		return p, domain.MatchTypeHardFP, score, nil
	}

	p, err = s.profiles.FindProfileBySoftFP(ctx, softFP)
	if err != nil {
		return nil, "", 0, err
	}
	if p != nil {
		score := s.computeSoftScore(input, p, softFP)
		if score >= s.threshold {
			return p, domain.MatchTypeSoftFP, score, nil
		}
	}

	candidates, err := s.profiles.FindCandidateProfiles(ctx, input.OSFamily, input.CPUCores, input.TimezoneName, input.ScreenWidth, input.ScreenHeight, input.MaxTouchPoints, candidateLimit)
	if err != nil {
		return nil, "", 0, err
	}

	var bestProfile *domain.BrowserProfile
	var bestScore float64

	for i := range candidates {
		c := &candidates[i]
		score := s.computeSoftScore(input, c, softFP)
		if score > bestScore {
			bestScore = score
			bestProfile = c
		}
	}

	if bestProfile != nil && bestScore >= s.threshold {
		return bestProfile, domain.MatchTypeCandidate, bestScore, nil
	}

	return nil, domain.MatchTypeNew, 0, nil
}

func (s *FingerprintService) fieldsFromInput(input domain.FingerprintInput, hardFP, softFP string) fingerprint.Fields {
	f := fingerprint.Fields{
		DeviceID: input.DeviceID,
		HardFP:   hardFP,

		AudioFP:             input.AudioFP,
		FontsFP:             input.FontsFP,
		WebGLParamsHash:     input.WebGLParamsHash,
		WebGLExtensionsHash: input.WebGLExtensionsHash,
		MediaCodecsHash:     input.MediaCodecsHash,
		MathFP:              input.MathFP,
		UserAgentFamily:     input.UserAgentFamily,
		OSFamily:            input.OSFamily,

		SoftFP:         softFP,
		CPUCores:       input.CPUCores,
		DeviceMemoryGB: input.DeviceMemoryGB,
		TimezoneName:   input.TimezoneName,
		Languages:      input.Languages,
		ScreenWidth:    input.ScreenWidth,
		ScreenHeight:   input.ScreenHeight,
		PixelRatio:     input.PixelRatio,
		MaxTouchPoints: input.MaxTouchPoints,
		ColorDepth:     input.ColorDepth,
		Platform:       input.Platform,

		AvailScreenWidth:  input.AvailScreenWidth,
		AvailScreenHeight: input.AvailScreenHeight,
		ColorGamut:        input.ColorGamut,
		HDR:               input.HDR,
		DoNotTrack:        input.DoNotTrack,
		PDFViewer:         input.PDFViewerEnabled,
		ForcedColors:      input.ForcedColors,
		TimezoneOffset:    input.TimezoneOffset,
		IntlLocale:        input.IntlLocale,
	}

	if input.MediaDevicesCount != nil {
		f.MediaDevicesAudio = &input.MediaDevicesCount.AudioInput
		f.MediaDevicesVideo = &input.MediaDevicesCount.VideoInput
		f.MediaDevicesOut = &input.MediaDevicesCount.AudioOutput
	}

	return f
}

func (s *FingerprintService) fieldsFromProfile(p *domain.BrowserProfile) fingerprint.Fields {
	f := fingerprint.Fields{
		DeviceID: p.CurrentDeviceID,
		HardFP:   p.CurrentHardFP,

		AudioFP:             p.AudioFP,
		FontsFP:             p.FontsFP,
		WebGLParamsHash:     p.WebGLParamsHash,
		WebGLExtensionsHash: p.WebGLExtensionsHash,
		MediaCodecsHash:     p.MediaCodecsHash,
		MathFP:              p.MathFP,
		UserAgentFamily:     p.UserAgentFamily,
		OSFamily:            p.OSFamily,

		SoftFP:         p.CurrentSoftFP,
		CPUCores:       p.CPUCores,
		DeviceMemoryGB: p.DeviceMemoryGB,
		TimezoneName:   p.TimezoneName,
		Languages:      p.Languages,
		ScreenWidth:    p.ScreenWidth,
		ScreenHeight:   p.ScreenHeight,
		PixelRatio:     p.PixelRatio,
		MaxTouchPoints: p.MaxTouchPoints,
		ColorDepth:     p.ColorDepth,
		Platform:       p.Platform,
	}

	var weak weakAttrs
	if len(p.VariableAttrs) > 0 {
		_ = json.Unmarshal(p.VariableAttrs, &weak)
	}
	f.AvailScreenWidth = weak.AvailScreenWidth
	f.AvailScreenHeight = weak.AvailScreenHeight
	f.ColorGamut = weak.ColorGamut
	f.HDR = weak.HDR
	f.DoNotTrack = weak.DoNotTrack
	f.PDFViewer = weak.PDFViewer
	f.ForcedColors = weak.ForcedColors
	f.TimezoneOffset = weak.TimezoneOffset
	f.IntlLocale = weak.IntlLocale
	if weak.MediaDevicesAudio != nil {
		f.MediaDevicesAudio = weak.MediaDevicesAudio
	}
	if weak.MediaDevicesVideo != nil {
		f.MediaDevicesVideo = weak.MediaDevicesVideo
	}
	if weak.MediaDevicesOut != nil {
		f.MediaDevicesOut = weak.MediaDevicesOut
	}

	return f
}

func (s *FingerprintService) computeFullScore(input domain.FingerprintInput, profile *domain.BrowserProfile, hardFP, softFP string) float64 {
	return fingerprint.ComputeFullSimilarity(s.fieldsFromInput(input, hardFP, softFP), s.fieldsFromProfile(profile), s.weights)
}

func (s *FingerprintService) computeSoftScore(input domain.FingerprintInput, profile *domain.BrowserProfile, softFP string) float64 {
	return fingerprint.ComputeSoftSimilarity(s.fieldsFromInput(input, "", softFP), s.fieldsFromProfile(profile), s.weights)
}

func (s *FingerprintService) updateProfile(p *domain.BrowserProfile, input domain.FingerprintInput, hardFP, softFP, hardwareFP string, clusterID int64, observedAt time.Time) {
	p.DeviceClusterID = &clusterID
	p.HardwareFP = hardwareFP
	p.CurrentDeviceID = input.DeviceID
	p.CurrentCanvasFP = input.CanvasFP
	p.CurrentWebGLVendor = input.WebGLVendor
	p.CurrentWebGLRenderer = input.WebGLRenderer
	p.CurrentHardFP = hardFP
	p.CurrentSoftFP = softFP

	p.AudioFP = input.AudioFP
	p.FontsFP = input.FontsFP
	p.WebGLParamsHash = input.WebGLParamsHash
	p.WebGLExtensionsHash = input.WebGLExtensionsHash
	p.MathFP = input.MathFP
	p.MediaCodecsHash = input.MediaCodecsHash

	p.UserAgentFamily = input.UserAgentFamily
	p.OSFamily = input.OSFamily
	p.CPUCores = input.CPUCores
	p.DeviceMemoryGB = input.DeviceMemoryGB
	p.TimezoneName = input.TimezoneName
	p.LanguageCode = input.LanguageCode
	p.Languages = input.Languages
	p.ScreenWidth = input.ScreenWidth
	p.ScreenHeight = input.ScreenHeight
	p.PixelRatio = input.PixelRatio
	p.ColorDepth = input.ColorDepth
	p.MaxTouchPoints = input.MaxTouchPoints
	p.Platform = input.Platform

	p.VariableAttrs = buildWeakAttrsJSON(input)
	p.LastSeen = observedAt
	p.EventCount++
}

func (s *FingerprintService) buildNewProfile(input domain.FingerprintInput, hardFP, softFP, hardwareFP string, clusterID int64, observedAt time.Time) *domain.BrowserProfile {
	return &domain.BrowserProfile{
		DeviceClusterID:      &clusterID,
		HardwareFP:           hardwareFP,
		CurrentDeviceID:      input.DeviceID,
		CurrentCanvasFP:      input.CanvasFP,
		CurrentWebGLVendor:   input.WebGLVendor,
		CurrentWebGLRenderer: input.WebGLRenderer,
		CurrentHardFP:        hardFP,
		CurrentSoftFP:        softFP,

		AudioFP:             input.AudioFP,
		FontsFP:             input.FontsFP,
		WebGLParamsHash:     input.WebGLParamsHash,
		WebGLExtensionsHash: input.WebGLExtensionsHash,
		MathFP:              input.MathFP,
		MediaCodecsHash:     input.MediaCodecsHash,

		UserAgentFamily: input.UserAgentFamily,
		OSFamily:        input.OSFamily,
		CPUCores:        input.CPUCores,
		DeviceMemoryGB:  input.DeviceMemoryGB,
		TimezoneName:    input.TimezoneName,
		LanguageCode:    input.LanguageCode,
		Languages:       input.Languages,
		ScreenWidth:     input.ScreenWidth,
		ScreenHeight:    input.ScreenHeight,
		PixelRatio:      input.PixelRatio,
		ColorDepth:      input.ColorDepth,
		MaxTouchPoints:  input.MaxTouchPoints,
		Platform:        input.Platform,

		StableAttrs:       json.RawMessage(`{}`),
		VariableAttrs:     buildWeakAttrsJSON(input),
		FirstSeen:         observedAt,
		LastSeen:          observedAt,
		EventCount:        1,
		LinkedAccountsCnt: 0,
	}
}

func (s *FingerprintService) buildEvent(input domain.FingerprintInput, profileID int64, hardFP, softFP, hardwareFP string, observedAt time.Time) *domain.FingerprintEvent {
	e := &domain.FingerprintEvent{
		BrowserProfileID: &profileID,
		AccountID:        input.AccountID,
		ObservedAt:       observedAt,
		HardwareFP:       hardwareFP,

		DeviceID:      input.DeviceID,
		CanvasFP:      input.CanvasFP,
		WebGLVendor:   input.WebGLVendor,
		WebGLRenderer: input.WebGLRenderer,
		HardFP:        hardFP,
		SoftFP:        softFP,

		AudioFP:             input.AudioFP,
		FontsFP:             input.FontsFP,
		WebGLParamsHash:     input.WebGLParamsHash,
		WebGLExtensionsHash: input.WebGLExtensionsHash,
		MathFP:              input.MathFP,
		MediaCodecsHash:     input.MediaCodecsHash,

		UserAgentRaw:    input.UserAgentRaw,
		UserAgentFamily: input.UserAgentFamily,
		OSFamily:        input.OSFamily,
		CPUCores:        input.CPUCores,
		DeviceMemoryGB:  input.DeviceMemoryGB,
		TimezoneName:    input.TimezoneName,
		LanguageCode:    input.LanguageCode,
		Languages:       input.Languages,
		ScreenWidth:     input.ScreenWidth,
		ScreenHeight:    input.ScreenHeight,
		PixelRatio:      input.PixelRatio,
		ColorDepth:      input.ColorDepth,
		MaxTouchPoints:  input.MaxTouchPoints,
		Platform:        input.Platform,
		Attrs:           input.Attrs,
	}

	if input.IPAddr != "" {
		if addr, err := netip.ParseAddr(input.IPAddr); err == nil {
			e.IPAddr = &addr
		}
	}

	return e
}

type weakAttrs struct {
	AvailScreenWidth   *int    `json:"avail_screen_width,omitempty"`
	AvailScreenHeight  *int    `json:"avail_screen_height,omitempty"`
	DoNotTrack         *string `json:"do_not_track,omitempty"`
	PDFViewer          *bool   `json:"pdf_viewer_enabled,omitempty"`
	ColorGamut         string  `json:"color_gamut,omitempty"`
	HDR                *bool   `json:"hdr,omitempty"`
	ForcedColors       *bool   `json:"forced_colors,omitempty"`
	PrefersColorScheme string  `json:"prefers_color_scheme,omitempty"`
	MediaDevicesAudio  *int    `json:"media_devices_audioinput,omitempty"`
	MediaDevicesVideo  *int    `json:"media_devices_videoinput,omitempty"`
	MediaDevicesOut    *int    `json:"media_devices_audiooutput,omitempty"`
	TimezoneOffset     *int    `json:"timezone_offset,omitempty"`
	IntlLocale         string  `json:"intl_locale,omitempty"`
}

func buildWeakAttrsJSON(input domain.FingerprintInput) json.RawMessage {
	w := weakAttrs{
		AvailScreenWidth:   input.AvailScreenWidth,
		AvailScreenHeight:  input.AvailScreenHeight,
		DoNotTrack:         input.DoNotTrack,
		PDFViewer:          input.PDFViewerEnabled,
		ColorGamut:         input.ColorGamut,
		HDR:                input.HDR,
		ForcedColors:       input.ForcedColors,
		PrefersColorScheme: input.PrefersColorScheme,
		TimezoneOffset:     input.TimezoneOffset,
		IntlLocale:         input.IntlLocale,
	}
	if input.MediaDevicesCount != nil {
		w.MediaDevicesAudio = &input.MediaDevicesCount.AudioInput
		w.MediaDevicesVideo = &input.MediaDevicesCount.VideoInput
		w.MediaDevicesOut = &input.MediaDevicesCount.AudioOutput
	}
	data, err := json.Marshal(w)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}
