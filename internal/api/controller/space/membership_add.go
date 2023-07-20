// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

type MembershipAddInput struct {
	UserUID string              `json:"user_uid"`
	Role    enum.MembershipRole `json:"role"`
}

func (in *MembershipAddInput) Validate() error {
	if in.UserUID == "" {
		return usererror.BadRequest("UserUID must be provided")
	}

	if in.Role == "" {
		return usererror.BadRequest("Role must be provided")
	}

	role, ok := in.Role.Sanitize()
	if !ok {
		msg := fmt.Sprintf("Provided role '%s' is not suppored. Valid values are: %v",
			in.Role, enum.MembershipRoles)
		return usererror.BadRequest(msg)
	}

	in.Role = role

	return nil
}

// MembershipAdd adds a new membership to a space.
func (c *Controller) MembershipAdd(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *MembershipAddInput,
) (*types.Membership, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit, false); err != nil {
		return nil, err
	}

	err = in.Validate()
	if err != nil {
		return nil, err
	}

	user, err := c.principalStore.FindUserByUID(ctx, in.UserUID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return nil, usererror.BadRequestf("User '%s' not found", in.UserUID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to find the user: %w", err)
	}

	now := time.Now().UnixMilli()

	membership := &types.Membership{
		SpaceID:     space.ID,
		PrincipalID: user.ID,
		CreatedBy:   session.Principal.ID,
		Created:     now,
		Updated:     now,
		Role:        in.Role,

		Principal: *user.ToPrincipalInfo(),
		AdddedBy:  *session.Principal.ToPrincipalInfo(),
	}

	err = c.membershipStore.Create(ctx, membership)
	if err != nil {
		return nil, fmt.Errorf("failed to create new membership: %w", err)
	}

	return membership, nil
}
