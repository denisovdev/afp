package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dxngee/antifraud-processing/internal/domain"
)

type ClusterRepo struct {
	pool *pgxpool.Pool
}

func NewClusterRepo(pool *pgxpool.Pool) *ClusterRepo {
	return &ClusterRepo{pool: pool}
}

func (r *ClusterRepo) FindClusterByHardwareFP(ctx context.Context, hardwareFP string) (*domain.DeviceCluster, error) {
	q := getQuerier(ctx, r.pool)

	sql := `SELECT device_cluster_id, hardware_fp,
			COALESCE(webgl_vendor, ''), COALESCE(os_family, ''),
			cpu_cores, device_memory_gb, screen_width, screen_height,
			first_seen, last_seen, profiles_count, linked_accounts_cnt
		FROM device_clusters
		WHERE hardware_fp = $1
		LIMIT 1`

	var c domain.DeviceCluster
	err := q.QueryRow(ctx, sql, hardwareFP).Scan(
		&c.DeviceClusterID, &c.HardwareFP,
		&c.WebGLVendor, &c.OSFamily,
		&c.CPUCores, &c.DeviceMemoryGB,
		&c.ScreenWidth, &c.ScreenHeight,
		&c.FirstSeen, &c.LastSeen,
		&c.ProfilesCount, &c.LinkedAccountsCnt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find cluster by hardware_fp: %w", err)
	}
	return &c, nil
}

func (r *ClusterRepo) InsertDeviceCluster(ctx context.Context, c *domain.DeviceCluster) (int64, error) {
	q := getQuerier(ctx, r.pool)

	sql := `INSERT INTO device_clusters (
			hardware_fp, webgl_vendor, os_family,
			cpu_cores, device_memory_gb, screen_width, screen_height,
			first_seen, last_seen, profiles_count, linked_accounts_cnt
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING device_cluster_id`

	var id int64
	err := q.QueryRow(ctx, sql,
		c.HardwareFP, c.WebGLVendor, c.OSFamily,
		c.CPUCores, c.DeviceMemoryGB, c.ScreenWidth, c.ScreenHeight,
		c.FirstSeen, c.LastSeen, c.ProfilesCount, c.LinkedAccountsCnt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert device cluster: %w", err)
	}
	return id, nil
}

func (r *ClusterRepo) UpdateDeviceCluster(ctx context.Context, c *domain.DeviceCluster) error {
	q := getQuerier(ctx, r.pool)

	sql := `UPDATE device_clusters SET
			webgl_vendor = $2,
			os_family = $3,
			cpu_cores = $4,
			device_memory_gb = $5,
			screen_width = $6,
			screen_height = $7,
			last_seen = $8,
			profiles_count = $9,
			linked_accounts_cnt = $10
		WHERE device_cluster_id = $1`

	_, err := q.Exec(ctx, sql,
		c.DeviceClusterID,
		c.WebGLVendor, c.OSFamily,
		c.CPUCores, c.DeviceMemoryGB,
		c.ScreenWidth, c.ScreenHeight,
		c.LastSeen, c.ProfilesCount, c.LinkedAccountsCnt,
	)
	if err != nil {
		return fmt.Errorf("update device cluster: %w", err)
	}
	return nil
}

func (r *ClusterRepo) CountClusterLinkedAccounts(ctx context.Context, clusterID int64) (int, error) {
	q := getQuerier(ctx, r.pool)

	sql := `SELECT COUNT(DISTINCT apl.account_id)
		FROM account_profile_links apl
		JOIN browser_profiles bp ON bp.browser_profile_id = apl.browser_profile_id
		WHERE bp.device_cluster_id = $1`

	var count int
	if err := q.QueryRow(ctx, sql, clusterID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count cluster linked accounts: %w", err)
	}
	return count, nil
}

func (r *ClusterRepo) CountClusterProfiles(ctx context.Context, clusterID int64) (int, error) {
	q := getQuerier(ctx, r.pool)

	sql := `SELECT COUNT(*) FROM browser_profiles WHERE device_cluster_id = $1`

	var count int
	if err := q.QueryRow(ctx, sql, clusterID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count cluster profiles: %w", err)
	}
	return count, nil
}
