ALTER TABLE fingerprint_events ALTER COLUMN account_id TYPE bigint USING account_id::bigint;

ALTER TABLE account_profile_links DROP CONSTRAINT account_profile_links_pkey;
ALTER TABLE account_profile_links ALTER COLUMN account_id TYPE bigint USING account_id::bigint;
ALTER TABLE account_profile_links ADD PRIMARY KEY (account_id, browser_profile_id);
