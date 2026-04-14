package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dxngee/antifraud-processing/internal/domain"
)

type ProfileRepo struct {
	pool *pgxpool.Pool
}

func NewProfileRepo(pool *pgxpool.Pool) *ProfileRepo {
	return &ProfileRepo{pool: pool}
}

var profileColumns = `
	browser_profile_id,
	public_id,
	device_cluster_id,
	COALESCE(hardware_fp, ''),
	COALESCE(current_device_id, ''),
	COALESCE(current_canvas_fp, ''),
	COALESCE(current_webgl_vendor, ''),
	COALESCE(current_webgl_renderer, ''),
	current_hard_fp,
	current_soft_fp,
	COALESCE(audio_fp, ''),
	COALESCE(fonts_fp, ''),
	COALESCE(webgl_params_hash, ''),
	COALESCE(webgl_extensions_hash, ''),
	COALESCE(math_fp, ''),
	COALESCE(media_codecs_hash, ''),
	COALESCE(user_agent_family, ''),
	COALESCE(os_family, ''),
	cpu_cores, device_memory_gb,
	COALESCE(timezone_name, ''),
	COALESCE(language_code, ''),
	languages,
	screen_width, screen_height, pixel_ratio,
	color_depth, max_touch_points,
	COALESCE(platform, ''),
	COALESCE(stable_attrs, '{}'::jsonb),
	COALESCE(variable_attrs, '{}'::jsonb),
	first_seen, last_seen, event_count, linked_accounts_cnt
`

func scanProfile(row pgx.Row) (*domain.BrowserProfile, error) {
	var p domain.BrowserProfile
	err := row.Scan(
		&p.BrowserProfileID, &p.PublicID, &p.DeviceClusterID, &p.HardwareFP,
		&p.CurrentDeviceID, &p.CurrentCanvasFP,
		&p.CurrentWebGLVendor, &p.CurrentWebGLRenderer,
		&p.CurrentHardFP, &p.CurrentSoftFP,
		&p.AudioFP, &p.FontsFP, &p.WebGLParamsHash, &p.WebGLExtensionsHash,
		&p.MathFP, &p.MediaCodecsHash,
		&p.UserAgentFamily, &p.OSFamily,
		&p.CPUCores, &p.DeviceMemoryGB,
		&p.TimezoneName, &p.LanguageCode, &p.Languages,
		&p.ScreenWidth, &p.ScreenHeight, &p.PixelRatio,
		&p.ColorDepth, &p.MaxTouchPoints, &p.Platform,
		&p.StableAttrs, &p.VariableAttrs,
		&p.FirstSeen, &p.LastSeen, &p.EventCount, &p.LinkedAccountsCnt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan profile: %w", err)
	}
	return &p, nil
}

func (r *ProfileRepo) FindProfileByDeviceID(ctx context.Context, deviceID string) (*domain.BrowserProfile, error) {
	q := getQuerier(ctx, r.pool)
	sql := `SELECT ` + profileColumns + ` FROM browser_profiles WHERE current_device_id = $1 LIMIT 1`
	return scanProfile(q.QueryRow(ctx, sql, deviceID))
}

func (r *ProfileRepo) FindProfileByHardFP(ctx context.Context, hardFP string) (*domain.BrowserProfile, error) {
	q := getQuerier(ctx, r.pool)
	sql := `SELECT ` + profileColumns + ` FROM browser_profiles WHERE current_hard_fp = $1 LIMIT 1`
	return scanProfile(q.QueryRow(ctx, sql, hardFP))
}

func (r *ProfileRepo) FindProfileBySoftFP(ctx context.Context, softFP string) (*domain.BrowserProfile, error) {
	q := getQuerier(ctx, r.pool)
	sql := `SELECT ` + profileColumns + ` FROM browser_profiles WHERE current_soft_fp = $1 LIMIT 1`
	return scanProfile(q.QueryRow(ctx, sql, softFP))
}

func (r *ProfileRepo) FindCandidateProfiles(ctx context.Context, osFamily string, cpuCores *int, timezoneName string, screenWidth, screenHeight, maxTouchPoints *int, limit int) ([]domain.BrowserProfile, error) {
	q := getQuerier(ctx, r.pool)
	sql := `SELECT ` + profileColumns + `
		FROM browser_profiles
		WHERE os_family = $1
		  AND ($2::integer IS NULL OR cpu_cores = $2)
		  AND timezone_name = $3
		  AND ($4::integer IS NULL OR screen_width = $4)
		  AND ($5::integer IS NULL OR screen_height = $5)
		  AND ($6::integer IS NULL OR max_touch_points = $6)
		ORDER BY last_seen DESC
		LIMIT $7`

	rows, err := q.Query(ctx, sql, osFamily, cpuCores, timezoneName, screenWidth, screenHeight, maxTouchPoints, limit)
	if err != nil {
		return nil, fmt.Errorf("find candidates: %w", err)
	}
	defer rows.Close()

	var profiles []domain.BrowserProfile
	for rows.Next() {
		var p domain.BrowserProfile
		if err := rows.Scan(
			&p.BrowserProfileID, &p.PublicID, &p.DeviceClusterID, &p.HardwareFP,
			&p.CurrentDeviceID, &p.CurrentCanvasFP,
			&p.CurrentWebGLVendor, &p.CurrentWebGLRenderer,
			&p.CurrentHardFP, &p.CurrentSoftFP,
			&p.AudioFP, &p.FontsFP, &p.WebGLParamsHash, &p.WebGLExtensionsHash,
			&p.MathFP, &p.MediaCodecsHash,
			&p.UserAgentFamily, &p.OSFamily,
			&p.CPUCores, &p.DeviceMemoryGB,
			&p.TimezoneName, &p.LanguageCode, &p.Languages,
			&p.ScreenWidth, &p.ScreenHeight, &p.PixelRatio,
			&p.ColorDepth, &p.MaxTouchPoints, &p.Platform,
			&p.StableAttrs, &p.VariableAttrs,
			&p.FirstSeen, &p.LastSeen, &p.EventCount, &p.LinkedAccountsCnt,
		); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *ProfileRepo) InsertBrowserProfile(ctx context.Context, p *domain.BrowserProfile) (int64, string, error) {
	q := getQuerier(ctx, r.pool)

	stableAttrs := ensureJSON(p.StableAttrs)
	variableAttrs := ensureJSON(p.VariableAttrs)

	sql := `INSERT INTO browser_profiles (
		device_cluster_id, hardware_fp,
		current_device_id, current_canvas_fp,
		current_webgl_vendor, current_webgl_renderer,
		current_hard_fp, current_soft_fp,
		audio_fp, fonts_fp, webgl_params_hash, webgl_extensions_hash,
		math_fp, media_codecs_hash,
		user_agent_family, os_family,
		cpu_cores, device_memory_gb,
		timezone_name, language_code, languages,
		screen_width, screen_height, pixel_ratio,
		color_depth, max_touch_points, platform,
		stable_attrs, variable_attrs,
		first_seen, last_seen, event_count, linked_accounts_cnt
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,
		$31,$32,$33
	) RETURNING browser_profile_id, public_id`

	var id int64
	var publicID string
	err := q.QueryRow(ctx, sql,
		p.DeviceClusterID, p.HardwareFP,
		p.CurrentDeviceID, p.CurrentCanvasFP,
		p.CurrentWebGLVendor, p.CurrentWebGLRenderer,
		p.CurrentHardFP, p.CurrentSoftFP,
		p.AudioFP, p.FontsFP, p.WebGLParamsHash, p.WebGLExtensionsHash,
		p.MathFP, p.MediaCodecsHash,
		p.UserAgentFamily, p.OSFamily,
		p.CPUCores, p.DeviceMemoryGB,
		p.TimezoneName, p.LanguageCode, p.Languages,
		p.ScreenWidth, p.ScreenHeight, p.PixelRatio,
		p.ColorDepth, p.MaxTouchPoints, p.Platform,
		stableAttrs, variableAttrs,
		p.FirstSeen, p.LastSeen, p.EventCount, p.LinkedAccountsCnt,
	).Scan(&id, &publicID)
	if err != nil {
		return 0, "", fmt.Errorf("insert profile: %w", err)
	}
	return id, publicID, nil
}

func (r *ProfileRepo) UpdateBrowserProfile(ctx context.Context, p *domain.BrowserProfile) error {
	q := getQuerier(ctx, r.pool)

	stableAttrs := ensureJSON(p.StableAttrs)
	variableAttrs := ensureJSON(p.VariableAttrs)

	sql := `UPDATE browser_profiles SET
		device_cluster_id = $2,
		hardware_fp = $3,
		current_device_id = $4,
		current_canvas_fp = $5,
		current_webgl_vendor = $6,
		current_webgl_renderer = $7,
		current_hard_fp = $8,
		current_soft_fp = $9,
		audio_fp = $10,
		fonts_fp = $11,
		webgl_params_hash = $12,
		webgl_extensions_hash = $13,
		math_fp = $14,
		media_codecs_hash = $15,
		user_agent_family = $16,
		os_family = $17,
		cpu_cores = $18,
		device_memory_gb = $19,
		timezone_name = $20,
		language_code = $21,
		languages = $22,
		screen_width = $23,
		screen_height = $24,
		pixel_ratio = $25,
		color_depth = $26,
		max_touch_points = $27,
		platform = $28,
		stable_attrs = $29,
		variable_attrs = $30,
		last_seen = $31,
		event_count = $32,
		linked_accounts_cnt = $33
	WHERE browser_profile_id = $1`

	_, err := q.Exec(ctx, sql,
		p.BrowserProfileID,
		p.DeviceClusterID, p.HardwareFP,
		p.CurrentDeviceID, p.CurrentCanvasFP,
		p.CurrentWebGLVendor, p.CurrentWebGLRenderer,
		p.CurrentHardFP, p.CurrentSoftFP,
		p.AudioFP, p.FontsFP, p.WebGLParamsHash, p.WebGLExtensionsHash,
		p.MathFP, p.MediaCodecsHash,
		p.UserAgentFamily, p.OSFamily,
		p.CPUCores, p.DeviceMemoryGB,
		p.TimezoneName, p.LanguageCode, p.Languages,
		p.ScreenWidth, p.ScreenHeight, p.PixelRatio,
		p.ColorDepth, p.MaxTouchPoints, p.Platform,
		stableAttrs, variableAttrs,
		p.LastSeen, p.EventCount, p.LinkedAccountsCnt,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func ensureJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
