package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepo struct {
	pool *pgxpool.Pool
}

func NewLinkRepo(pool *pgxpool.Pool) *LinkRepo {
	return &LinkRepo{pool: pool}
}

func (r *LinkRepo) UpsertAccountProfileLink(ctx context.Context, accountID, profileID int64) error {
	q := getQuerier(ctx, r.pool)

	sql := `INSERT INTO account_profile_links (account_id, browser_profile_id, first_seen, last_seen, events_count)
		VALUES ($1, $2, now(), now(), 1)
		ON CONFLICT (account_id, browser_profile_id)
		DO UPDATE SET last_seen = now(), events_count = account_profile_links.events_count + 1`

	_, err := q.Exec(ctx, sql, accountID, profileID)
	if err != nil {
		return fmt.Errorf("upsert account link: %w", err)
	}
	return nil
}

func (r *LinkRepo) CountLinkedAccounts(ctx context.Context, profileID int64) (int, error) {
	q := getQuerier(ctx, r.pool)

	var cnt int
	err := q.QueryRow(ctx,
		`SELECT count(*) FROM account_profile_links WHERE browser_profile_id = $1`,
		profileID,
	).Scan(&cnt)
	if err != nil {
		return 0, fmt.Errorf("count linked accounts: %w", err)
	}
	return cnt, nil
}

