package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dxngee/antifraud-processing/internal/domain"
)

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) InsertFingerprintEvent(ctx context.Context, e *domain.FingerprintEvent) (int64, error) {
	q := getQuerier(ctx, r.pool)

	attrs := ensureJSON(e.Attrs)

	var ipStr *string
	if e.IPAddr != nil {
		s := e.IPAddr.String()
		ipStr = &s
	}

	sql := `INSERT INTO fingerprint_events (
		browser_profile_id, account_id, observed_at,
		hardware_fp,
		device_id, canvas_fp, webgl_vendor, webgl_renderer,
		hard_fp, soft_fp,
		audio_fp, fonts_fp, webgl_params_hash, webgl_extensions_hash,
		math_fp, media_codecs_hash,
		user_agent_raw, user_agent_family, os_family,
		cpu_cores, device_memory_gb,
		timezone_name, language_code, languages,
		screen_width, screen_height, pixel_ratio,
		color_depth, max_touch_points, platform,
		ip_addr, attrs
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,
		$31::inet,$32
	) RETURNING event_id`

	var id int64
	err := q.QueryRow(ctx, sql,
		e.BrowserProfileID, e.AccountID, e.ObservedAt,
		e.HardwareFP,
		e.DeviceID, e.CanvasFP, e.WebGLVendor, e.WebGLRenderer,
		e.HardFP, e.SoftFP,
		e.AudioFP, e.FontsFP, e.WebGLParamsHash, e.WebGLExtensionsHash,
		e.MathFP, e.MediaCodecsHash,
		e.UserAgentRaw, e.UserAgentFamily, e.OSFamily,
		e.CPUCores, e.DeviceMemoryGB,
		e.TimezoneName, e.LanguageCode, e.Languages,
		e.ScreenWidth, e.ScreenHeight, e.PixelRatio,
		e.ColorDepth, e.MaxTouchPoints, e.Platform,
		ipStr, attrs,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert event: %w", err)
	}
	return id, nil
}
