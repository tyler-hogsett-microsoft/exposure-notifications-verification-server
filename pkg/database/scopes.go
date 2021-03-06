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

package database

import (
	"github.com/google/exposure-notifications-verification-server/internal/project"
	"github.com/jinzhu/gorm"
)

// Scope is a type alias to a gorm scope. It exists to reduce duplicate and
// function length. Note this is an ALIAS. It is NOT a new type.
type Scope = func(db *gorm.DB) *gorm.DB

// OnlySystemAdmins returns a scope that restricts the query to system admins.
// It's only applicable to functions that query User.
func OnlySystemAdmins() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(&User{SystemAdmin: true})
	}
}

// WithUserSearch returns a scope that adds querying for users by email and
// name, case-insensitive. It's only applicable to functions that query User.
func WithUserSearch(q string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		q = project.TrimSpace(q)
		if q != "" {
			q = `%` + q + `%`
			return db.Where("users.email ILIKE ? OR users.name ILIKE ?", q, q)
		}
		return db
	}
}

// WithAuditTime returns a scope that adds querying for Audit events by time.
func WithAuditTime(from, to string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		from = project.TrimSpace(from)
		if from != "" {
			db = db.Where("audit_entries.created_at >= ?", from)
		}

		to = project.TrimSpace(to)
		if to != "" {
			db = db.Where("audit_entries.created_at <= ?", to)
		}
		return db
	}
}

// WithAuditRealmID returns a scope that adds querying for Audit events by
// realm. The provided ID is expected to be stringable (int, uint, string).
func WithAuditRealmID(id uint) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("audit_entries.realm_id = ?", id)
	}
}

// WithRealmSearch returns a scope that adds querying for realms by name. It's
// only applicable to functions that query Realm.
func WithRealmSearch(q string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		q = project.TrimSpace(q)
		if q != "" {
			q = `%` + q + `%`
			return db.Where("realms.name ILIKE ?", q)
		}
		return db
	}
}

// WithoutAuditTest excludes audit entries related to test entries created from
// SystemTest.
func WithoutAuditTest() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("audit_entries.actor_id != ?", SystemTest.AuditID())
	}
}

// WithAppOS returns a scope that for querying MobileApps by Operating System type.
func WithAppOS(os OSType) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("os = ?", os)
	}
}

// WithAuthorizedAppSearch returns a scope that adds querying for API keys by
// name and preview, case-insensitive. It's only applicable to functions that
// query AuthorizedApp.
func WithAuthorizedAppSearch(q string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		q = project.TrimSpace(q)
		if q != "" {
			q = `%` + q + `%`
			return db.Where("authorized_apps.name ILIKE ? OR authorized_apps.api_key_preview ILIKE ?", q, q)
		}
		return db
	}
}

// WithMobileAppSearch returns a scope that adds querying for mobile apps by
// name, case-insensitive. It's only applicable to functions that query
// MobileApp.
func WithMobileAppSearch(q string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		q = project.TrimSpace(q)
		if q != "" {
			q = `%` + q + `%`
			return db.Where("mobile_apps.name ILIKE ?", q)
		}
		return db
	}
}
