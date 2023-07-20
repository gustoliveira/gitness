// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/slices"
)

type PermissionCacheKey struct {
	PrincipalID int64
	SpaceRef    string
	Permission  enum.Permission
}
type PermissionCache cache.Cache[PermissionCacheKey, bool]

func NewPermissionCache(
	spaceStore store.SpaceStore,
	membershipStore store.MembershipStore,
	cacheDuration time.Duration,
) PermissionCache {
	return cache.New[PermissionCacheKey, bool](permissionCacheGetter{
		spaceStore:      spaceStore,
		membershipStore: membershipStore,
	}, cacheDuration)
}

type permissionCacheGetter struct {
	spaceStore      store.SpaceStore
	membershipStore store.MembershipStore
}

func (g permissionCacheGetter) Find(ctx context.Context, key PermissionCacheKey) (bool, error) {
	spaceRef := key.SpaceRef
	principalID := key.PrincipalID

	// Find the starting space.
	space, err := g.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return false, fmt.Errorf("failed to find space '%s': %w", spaceRef, err)
	}

	// limit the depth to be safe (e.g. root/space1/space2 => maxDepth of 3)
	maxDepth := len(paths.Segments(spaceRef))

	for depth := 0; depth < maxDepth; depth++ {
		// Find the membership in the current space.
		membership, err := g.membershipStore.Find(ctx, types.MembershipKey{
			SpaceID:     space.ID,
			PrincipalID: principalID,
		})
		if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return false, fmt.Errorf("failed to find membership: %w", err)
		}

		// If the membership is defined in the current space, check if the user has the required permission.
		if membership != nil {
			_, hasRole := slices.BinarySearch(membership.Role.Permissions(), key.Permission)
			if hasRole {
				return true, nil
			}
		}

		// If membership with the requested permission has not been found in the current space,
		// move to the parent space, if any.

		if space.ParentID == 0 {
			return false, nil
		}

		space, err = g.spaceStore.Find(ctx, space.ParentID)
		if err != nil {
			return false, fmt.Errorf("failed to find parent space with id %d: %w", space.ParentID, err)
		}
	}

	return false, nil
}
