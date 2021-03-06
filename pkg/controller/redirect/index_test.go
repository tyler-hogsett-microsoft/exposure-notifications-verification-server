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

package redirect_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/exposure-notifications-verification-server/internal/project"
	"github.com/google/exposure-notifications-verification-server/internal/routes"
	"github.com/google/exposure-notifications-verification-server/pkg/cache"
	"github.com/google/exposure-notifications-verification-server/pkg/config"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
)

func TestIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create cacher.
	cacher, err := cache.NewInMemory(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cacher.Close(); err != nil {
			t.Fatal(err)
		}
	})

	// Create database.
	testDatabaseInstance := database.MustTestInstance()
	t.Cleanup(func() {
		if err := testDatabaseInstance.Close(); err != nil {
			t.Fatal(err)
		}
	})
	db, _ := testDatabaseInstance.NewDatabase(t, cacher)

	// Create config.
	cfg := &config.RedirectConfig{
		AssetsPath: filepath.Join(project.Root(), "cmd", "enx-redirect", "assets"),
		DevMode:    true,
		HostnameConfig: map[string]string{
			"bad":  "nope",
			"okay": "bb",
		},
	}

	// Set realm to resolve.
	realm1, err := db.FindRealm(1)
	if err != nil {
		t.Fatal(err)
	}
	realm1.RegionCode = "aa"
	if err := db.SaveRealm(realm1, database.SystemTest); err != nil {
		t.Fatal(err)
	}

	// Create another realm with apps.
	realm2 := database.NewRealmWithDefaults("okay")
	realm2.RegionCode = "bb"
	if err := db.SaveRealm(realm2, database.SystemTest); err != nil {
		t.Fatal(err)
	}

	// Create iOS app
	iosApp := &database.MobileApp{
		Name:    "app1",
		RealmID: realm2.ID,
		URL:     "https://app1.example.com/",
		OS:      database.OSTypeIOS,
		AppID:   "com.example.app1",
	}
	if err := db.SaveMobileApp(iosApp, database.SystemTest); err != nil {
		t.Fatal(err)
	}

	// Create Android app
	app2 := &database.MobileApp{
		Name:    "app2",
		RealmID: realm2.ID,
		URL:     "https://app2.example.com/",
		OS:      database.OSTypeAndroid,
		AppID:   "com.example.app2",
		SHA:     "AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA:AA",
	}
	if err := db.SaveMobileApp(app2, database.SystemTest); err != nil {
		t.Fatal(err)
	}

	// Build routes.
	mux, err := routes.ENXRedirect(ctx, cfg, db, cacher)
	if err != nil {
		t.Fatal(err)
	}

	// Start server.
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
	})
	client := srv.Client()

	// Don't follow redirects.
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// Bad path
	t.Run("bad_path", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", srv.URL+"/css/view/main/gift.css", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if got, want := resp.StatusCode, 404; got != want {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected %d to be %d: %s", got, want, body)
		}
	})

	// No matching region returns a 404
	t.Run("no_matching_region", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", srv.URL, nil)
		req.Host = "not-real"
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if got, want := resp.StatusCode, 404; got != want {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected %d to be %d: %s", got, want, body)
		}
	})

	// A matching region that doesn't point to a realm returns 404
	t.Run("matching_region_no_realm", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", srv.URL, nil)
		req.Host = "bad"
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if got, want := resp.StatusCode, 404; got != want {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected %d to be %d: %s", got, want, body)
		}
	})

	// Not a mobile user agent returns a 404
	t.Run("not_mobile_user_agent", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", srv.URL, nil)
		req.Host = "okay"
		req.Header.Set("User-Agent", "bananarama")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if got, want := resp.StatusCode, 404; got != want {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected %d to be %d: %s", got, want, body)
		}
	})

	// Android redirects
	t.Run("android_redirect", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", srv.URL, nil)
		req.Host = "okay"
		req.Header.Set("User-Agent", "android")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if got, want := resp.StatusCode, 303; got != want {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected %d to be %d: %s", got, want, body)
		}

		exp := "intent:?r=BB#Intent;scheme=ens;package=com.example.app2;action=android.intent.action.VIEW;category=android.intent.category.BROWSABLE;S.browser_fallback_url=https%3A%2F%2Fapp2.example.com%2F;end"
		if got, want := resp.Header.Get("Location"), exp; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
	})

	// iOS redirects
	t.Run("ios_redirect", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequest("GET", srv.URL, nil)
		req.Host = "okay"
		req.Header.Set("User-Agent", "iphone")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if got, want := resp.StatusCode, 303; got != want {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected %d to be %d: %s", got, want, body)
		}

		if got, want := resp.Header.Get("Location"), "https://app1.example.com/"; got != want {
			t.Errorf("expected %q to be %q", got, want)
		}
	})
}
