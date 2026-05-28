package nats

import (
	"errors"
	"time"

	"router-sync/internal/models"
)

// ErrConflict is returned when a write loses an active/active conflict.
var ErrConflict = errors.New("write rejected: remote version is newer")

// ShouldAcceptWrite decides whether an incoming record should replace existing state.
// Active/active semantics: higher generation wins; then newer UpdatedAt; then lexicographic WriterID.
func ShouldAcceptWrite(existingGen uint64, existingUpdatedAt time.Time, existingWriterID string,
	incomingGen uint64, incomingUpdatedAt time.Time, incomingWriterID string) bool {
	if incomingGen > existingGen {
		return true
	}
	if incomingGen < existingGen {
		return false
	}
	if incomingUpdatedAt.After(existingUpdatedAt) {
		return true
	}
	if incomingUpdatedAt.Before(existingUpdatedAt) {
		return false
	}
	return incomingWriterID >= existingWriterID
}

// PrepareProviderWrite assigns writer metadata and generation for a new revision.
func PrepareProviderWrite(provider *models.InternetProvider, existing *models.InternetProvider, writerID string) {
	now := time.Now().UTC()
	provider.WriterID = writerID
	provider.UpdatedAt = now
	if existing == nil {
		if provider.Generation == 0 {
			provider.Generation = 1
		}
		if provider.CreatedAt.IsZero() {
			provider.CreatedAt = now
		}
		return
	}
	provider.Generation = existing.Generation + 1
}

// PreparePolicyWrite assigns writer metadata and generation for a new revision.
func PreparePolicyWrite(policy *models.RoutingPolicy, existing *models.RoutingPolicy, writerID string) {
	now := time.Now().UTC()
	policy.WriterID = writerID
	policy.UpdatedAt = now
	if existing == nil {
		if policy.Generation == 0 {
			policy.Generation = 1
		}
		if policy.CreatedAt.IsZero() {
			policy.CreatedAt = now
		}
		return
	}
	policy.Generation = existing.Generation + 1
}
