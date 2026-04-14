package repository

import (
	"context"

	"github.com/dxngee/antifraud-processing/internal/domain"
)

// ProfileRepository defines data-access operations for browser profiles.
type ProfileRepository interface {
	FindProfileByDeviceID(ctx context.Context, deviceID string) (*domain.BrowserProfile, error)
	FindProfileByHardFP(ctx context.Context, hardFP string) (*domain.BrowserProfile, error)
	FindProfileBySoftFP(ctx context.Context, softFP string) (*domain.BrowserProfile, error)
	FindCandidateProfiles(ctx context.Context, osFamily string, cpuCores *int, timezoneName string, screenWidth, screenHeight, maxTouchPoints *int, limit int) ([]domain.BrowserProfile, error)
	InsertBrowserProfile(ctx context.Context, p *domain.BrowserProfile) (int64, error)
	UpdateBrowserProfile(ctx context.Context, p *domain.BrowserProfile) error
}

// EventRepository defines data-access operations for fingerprint events.
type EventRepository interface {
	InsertFingerprintEvent(ctx context.Context, e *domain.FingerprintEvent) (int64, error)
}

// LinkRepository defines data-access operations for account-profile links.
type LinkRepository interface {
	UpsertAccountProfileLink(ctx context.Context, accountID, profileID int64) error
	CountLinkedAccounts(ctx context.Context, profileID int64) (int, error)
}

// ClusterRepository defines data-access operations for device clusters.
type ClusterRepository interface {
	FindClusterByHardwareFP(ctx context.Context, hardwareFP string) (*domain.DeviceCluster, error)
	InsertDeviceCluster(ctx context.Context, c *domain.DeviceCluster) (int64, error)
	UpdateDeviceCluster(ctx context.Context, c *domain.DeviceCluster) error
	CountClusterLinkedAccounts(ctx context.Context, clusterID int64) (int, error)
	CountClusterProfiles(ctx context.Context, clusterID int64) (int, error)
}

// TxManager wraps work inside a database transaction.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
