// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apikey

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
	"github.com/google/exposure-notifications-verification-server/pkg/rbac"
	"github.com/gorilla/mux"
)

// HandleUpdate handles an update.
func (c *Controller) HandleUpdate() http.Handler {
	type FormData struct {
		Name string `form:"name"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)

		session := controller.SessionFromContext(ctx)
		if session == nil {
			controller.MissingSession(w, r, c.h)
			return
		}
		flash := controller.Flash(session)

		membership := controller.MembershipFromContext(ctx)
		if membership == nil {
			controller.MissingMembership(w, r, c.h)
			return
		}
		if !membership.Can(rbac.APIKeyWrite) {
			controller.Unauthorized(w, r, c.h)
			return
		}

		currentRealm := membership.Realm
		currentUser := membership.User

		authApp, err := currentRealm.FindAuthorizedApp(c.db, vars["id"])
		if err != nil {
			if database.IsNotFound(err) {
				controller.Unauthorized(w, r, c.h)
				return
			}

			controller.InternalError(w, r, c.h, err)
			return
		}

		// Requested form, stop processing.
		if r.Method == http.MethodGet {
			c.renderEdit(ctx, w, authApp)
			return
		}

		var form FormData
		if err := controller.BindForm(w, r, &form); err != nil {
			authApp.Name = form.Name
			flash.Error("Failed to process form: %v", err)
			c.renderEdit(ctx, w, authApp)
		}

		// Build the authorized app struct
		authApp.Name = form.Name

		// Save
		if err := c.db.SaveAuthorizedApp(authApp, currentUser); err != nil {
			flash.Error("Failed to save api key: %v", err)
			c.renderEdit(ctx, w, authApp)
			return
		}

		flash.Alert("Successfully updated API key!")
		http.Redirect(w, r, fmt.Sprintf("/realm/apikeys/%d", authApp.ID), http.StatusSeeOther)
	})
}

// renderEdit renders the edit page.
func (c *Controller) renderEdit(ctx context.Context, w http.ResponseWriter, authApp *database.AuthorizedApp) {
	m := controller.TemplateMapFromContext(ctx)
	m.Title("Edit API key: %s", authApp.Name)
	m["authApp"] = authApp
	c.h.RenderHTML(w, "apikeys/edit", m)
}
