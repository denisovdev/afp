-- Medium-stability signals on browser_profiles
ALTER TABLE browser_profiles
    ADD COLUMN audio_fp              text,
    ADD COLUMN fonts_fp              text,
    ADD COLUMN webgl_params_hash     text,
    ADD COLUMN webgl_extensions_hash text,
    ADD COLUMN math_fp               text,
    ADD COLUMN media_codecs_hash     text;

-- Additional soft signals on browser_profiles
ALTER TABLE browser_profiles
    ADD COLUMN languages       text[],
    ADD COLUMN color_depth     integer,
    ADD COLUMN max_touch_points integer,
    ADD COLUMN platform        text;

-- Medium-stability signals on fingerprint_events
ALTER TABLE fingerprint_events
    ADD COLUMN audio_fp              text,
    ADD COLUMN fonts_fp              text,
    ADD COLUMN webgl_params_hash     text,
    ADD COLUMN webgl_extensions_hash text,
    ADD COLUMN math_fp               text,
    ADD COLUMN media_codecs_hash     text;

-- Additional soft signals on fingerprint_events
ALTER TABLE fingerprint_events
    ADD COLUMN languages       text[],
    ADD COLUMN color_depth     integer,
    ADD COLUMN max_touch_points integer,
    ADD COLUMN platform        text;

-- Indexes for medium-stability signals on profiles (used for candidate search)
CREATE INDEX idx_bp_audio_fp       ON browser_profiles (audio_fp)       WHERE audio_fp IS NOT NULL;
CREATE INDEX idx_bp_fonts_fp       ON browser_profiles (fonts_fp)       WHERE fonts_fp IS NOT NULL;
CREATE INDEX idx_bp_webgl_params   ON browser_profiles (webgl_params_hash) WHERE webgl_params_hash IS NOT NULL;

-- Extended composite index for candidate search
DROP INDEX IF EXISTS idx_bp_soft_combo;
CREATE INDEX idx_bp_soft_combo ON browser_profiles (os_family, cpu_cores, timezone_name, max_touch_points);
