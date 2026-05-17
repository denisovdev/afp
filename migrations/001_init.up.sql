CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Browser profiles: each row represents a recognized browser environment.
CREATE TABLE browser_profiles (
    browser_profile_id   bigserial       PRIMARY KEY,
    current_device_id    text,
    current_canvas_fp    text,
    current_webgl_vendor text,
    current_webgl_renderer text,
    current_hard_fp      text            NOT NULL,
    current_soft_fp      text            NOT NULL,
    user_agent_family    text,
    os_family            text,
    cpu_cores            integer,
    device_memory_gb     integer,
    timezone_name        text,
    language_code        text,
    screen_width         integer,
    screen_height        integer,
    pixel_ratio          numeric(6,2),
    stable_attrs         jsonb           NOT NULL DEFAULT '{}'::jsonb,
    variable_attrs       jsonb           NOT NULL DEFAULT '{}'::jsonb,
    first_seen           timestamptz     NOT NULL DEFAULT now(),
    last_seen            timestamptz     NOT NULL DEFAULT now(),
    event_count          integer         NOT NULL DEFAULT 0,
    linked_accounts_cnt  integer         NOT NULL DEFAULT 0
);

CREATE INDEX idx_bp_device_id       ON browser_profiles (current_device_id)   WHERE current_device_id IS NOT NULL;
CREATE INDEX idx_bp_canvas_fp       ON browser_profiles (current_canvas_fp)   WHERE current_canvas_fp IS NOT NULL;
CREATE INDEX idx_bp_hard_fp         ON browser_profiles (current_hard_fp);
CREATE INDEX idx_bp_soft_fp         ON browser_profiles (current_soft_fp);
CREATE INDEX idx_bp_soft_combo      ON browser_profiles (os_family, cpu_cores, timezone_name);
CREATE INDEX idx_bp_stable_attrs    ON browser_profiles USING gin (stable_attrs);
CREATE INDEX idx_bp_variable_attrs  ON browser_profiles USING gin (variable_attrs);
CREATE INDEX idx_bp_webgl_renderer  ON browser_profiles USING gin (current_webgl_renderer gin_trgm_ops);
CREATE INDEX idx_bp_ua_family_trgm  ON browser_profiles USING gin (user_agent_family gin_trgm_ops);

-- Raw fingerprint events coming from the client.
CREATE TABLE fingerprint_events (
    event_id             bigserial       PRIMARY KEY,
    browser_profile_id   bigint          REFERENCES browser_profiles(browser_profile_id) ON DELETE SET NULL,
    account_id           bigint          NOT NULL,
    observed_at          timestamptz     NOT NULL DEFAULT now(),
    device_id            text,
    canvas_fp            text,
    webgl_vendor         text,
    webgl_renderer       text,
    hard_fp              text            NOT NULL,
    soft_fp              text            NOT NULL,
    user_agent_raw       text,
    user_agent_family    text,
    os_family            text,
    cpu_cores            integer,
    device_memory_gb     integer,
    timezone_name        text,
    language_code        text,
    screen_width         integer,
    screen_height        integer,
    pixel_ratio          numeric(6,2),
    ip_addr              inet,
    attrs                jsonb           NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX idx_fe_account_id      ON fingerprint_events (account_id);
CREATE INDEX idx_fe_profile_id      ON fingerprint_events (browser_profile_id);
CREATE INDEX idx_fe_observed_at     ON fingerprint_events (observed_at DESC);
CREATE INDEX idx_fe_ip_addr         ON fingerprint_events (ip_addr);
CREATE INDEX idx_fe_attrs           ON fingerprint_events USING gin (attrs);

-- Links between accounts and browser profiles.
CREATE TABLE account_profile_links (
    account_id           bigint          NOT NULL,
    browser_profile_id   bigint          NOT NULL REFERENCES browser_profiles(browser_profile_id) ON DELETE CASCADE,
    first_seen           timestamptz     NOT NULL DEFAULT now(),
    last_seen            timestamptz     NOT NULL DEFAULT now(),
    events_count         integer         NOT NULL DEFAULT 1,
    PRIMARY KEY (account_id, browser_profile_id)
);
