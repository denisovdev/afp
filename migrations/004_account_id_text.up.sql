ALTER TABLE account_profile_links DROP CONSTRAINT account_profile_links_pkey;
ALTER TABLE account_profile_links ALTER COLUMN account_id TYPE text USING account_id::text;
ALTER TABLE account_profile_links ADD PRIMARY KEY (account_id, browser_profile_id);

ALTER TABLE fingerprint_events ALTER COLUMN account_id TYPE text USING account_id::text;
