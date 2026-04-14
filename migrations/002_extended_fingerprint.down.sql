DROP INDEX IF EXISTS idx_bp_soft_combo;
CREATE INDEX idx_bp_soft_combo ON browser_profiles (os_family, cpu_cores, timezone_name);

DROP INDEX IF EXISTS idx_bp_webgl_params;
DROP INDEX IF EXISTS idx_bp_fonts_fp;
DROP INDEX IF EXISTS idx_bp_audio_fp;

ALTER TABLE fingerprint_events
    DROP COLUMN IF EXISTS platform,
    DROP COLUMN IF EXISTS max_touch_points,
    DROP COLUMN IF EXISTS color_depth,
    DROP COLUMN IF EXISTS languages,
    DROP COLUMN IF EXISTS media_codecs_hash,
    DROP COLUMN IF EXISTS math_fp,
    DROP COLUMN IF EXISTS webgl_extensions_hash,
    DROP COLUMN IF EXISTS webgl_params_hash,
    DROP COLUMN IF EXISTS fonts_fp,
    DROP COLUMN IF EXISTS audio_fp;

ALTER TABLE browser_profiles
    DROP COLUMN IF EXISTS platform,
    DROP COLUMN IF EXISTS max_touch_points,
    DROP COLUMN IF EXISTS color_depth,
    DROP COLUMN IF EXISTS languages,
    DROP COLUMN IF EXISTS media_codecs_hash,
    DROP COLUMN IF EXISTS math_fp,
    DROP COLUMN IF EXISTS webgl_extensions_hash,
    DROP COLUMN IF EXISTS webgl_params_hash,
    DROP COLUMN IF EXISTS fonts_fp,
    DROP COLUMN IF EXISTS audio_fp;
