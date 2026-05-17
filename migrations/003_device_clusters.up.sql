CREATE TABLE device_clusters (
    device_cluster_id   bigserial       PRIMARY KEY,
    hardware_fp         text            NOT NULL UNIQUE,
    webgl_vendor        text,
    os_family           text,
    cpu_cores           integer,
    device_memory_gb    integer,
    screen_width        integer,
    screen_height       integer,
    first_seen          timestamptz     NOT NULL DEFAULT now(),
    last_seen           timestamptz     NOT NULL DEFAULT now(),
    profiles_count      integer         NOT NULL DEFAULT 0,
    linked_accounts_cnt integer         NOT NULL DEFAULT 0
);

CREATE INDEX idx_dc_hardware_fp ON device_clusters (hardware_fp);

ALTER TABLE browser_profiles
    ADD COLUMN device_cluster_id bigint REFERENCES device_clusters(device_cluster_id) ON DELETE SET NULL,
    ADD COLUMN hardware_fp text;

CREATE INDEX idx_bp_hardware_fp    ON browser_profiles (hardware_fp) WHERE hardware_fp IS NOT NULL;
CREATE INDEX idx_bp_cluster_id     ON browser_profiles (device_cluster_id) WHERE device_cluster_id IS NOT NULL;

ALTER TABLE fingerprint_events
    ADD COLUMN hardware_fp text;
