// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleMembershipAdd handles API that adds a new membership to a space.
func HandleMembershipAdd(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		in := new(space.MembershipAddInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		memberInfo, err := spaceCtrl.MembershipAdd(ctx, session, spaceRef, in)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusCreated, memberInfo)
	}
}
