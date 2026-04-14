ALTER TABLE browser_profiles
    ADD COLUMN public_id uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE;

ALTER TABLE device_clusters
    ADD COLUMN public_id uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE;
