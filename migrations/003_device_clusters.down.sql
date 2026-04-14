ALTER TABLE fingerprint_events DROP COLUMN IF EXISTS hardware_fp;

DROP INDEX IF EXISTS idx_bp_cluster_id;
DROP INDEX IF EXISTS idx_bp_hardware_fp;

ALTER TABLE browser_profiles DROP COLUMN IF EXISTS hardware_fp;
ALTER TABLE browser_profiles DROP COLUMN IF EXISTS device_cluster_id;

DROP TABLE IF EXISTS device_clusters;
