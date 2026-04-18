// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"tailscale.com/client/tailscale/v2"
)

// userInvite is the Tailscale API user invite payload (subset we need).
type userInvite struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// membershipAPIClient wraps tailscale.Client to add UserInvites and user action APIs
// that are not exposed by the v2 client. Uses the same HTTP client and auth as the
// tailscale.Client (see research.md T001).
type membershipAPIClient struct {
	*tailscale.Client
}

func membershipAPI(c *tailscale.Client) *membershipAPIClient {
	return &membershipAPIClient{Client: c}
}

func (m *membershipAPIClient) baseURL() *url.URL {
	if m.Client.BaseURL != nil {
		return m.Client.BaseURL
	}
	u, _ := url.Parse("https://api.tailscale.com")
	return u
}

func (m *membershipAPIClient) do(ctx context.Context, method, rawURL string, body any) (*http.Response, error) {
	// Trigger the v2 client's lazy init so that:
	//   - For OAuth / Federated Identity (m.Client.Auth != nil), m.Client.HTTP is
	//     replaced with the auth-decorated *http.Client returned by
	//     m.Client.Auth.HTTPClient(...) and m.Client.APIKey is zeroed.
	//   - For API-key mode (m.Client.Auth == nil), m.Client.HTTP is initialised to
	//     a plain *http.Client (1m timeout) and m.Client.APIKey is preserved.
	// Any v2 resource accessor triggers init via sync.Once; Users() is the
	// closest to the membership domain. See specs/002-standalone-membership-provider/research.md §R1.
	_ = m.Client.Users()

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if m.Client.UserAgent != "" {
		req.Header.Set("User-Agent", m.Client.UserAgent)
	}
	// API-key mode: the v2 client adds Basic auth in its own buildRequest, but
	// this helper builds requests directly, so we set it here. After init() this
	// branch only fires when no Auth was configured (Auth != nil zeroes APIKey).
	if m.Client.APIKey != "" {
		req.SetBasicAuth(m.Client.APIKey, "")
	}
	return m.Client.HTTP.Do(req)
}

// listUserInvites returns all open user invites for the tailnet.
func (m *membershipAPIClient) listUserInvites(ctx context.Context) ([]userInvite, error) {
	path := fmt.Sprintf("%s/api/v2/tailnet/%s/user-invites", m.baseURL().String(), url.PathEscape(m.Client.Tailnet))
	resp, err := m.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list user invites: %s (%d): %s", resp.Status, resp.StatusCode, string(body))
	}
	var list []userInvite
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// createUserInvite creates a user invite with the given email and role.
// API expects POST body as array of invites; we send one.
func (m *membershipAPIClient) createUserInvite(ctx context.Context, email, role string) (*userInvite, error) {
	path := fmt.Sprintf("%s/api/v2/tailnet/%s/user-invites", m.baseURL().String(), url.PathEscape(m.Client.Tailnet))
	body := []map[string]string{{"email": email, "role": role}}
	resp, err := m.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create user invite: %s (%d): %s", resp.Status, resp.StatusCode, string(bodyBytes))
	}
	var list []userInvite
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("create user invite: empty response")
	}
	return &list[0], nil
}

// deleteUserInvite deletes a user invite by ID.
func (m *membershipAPIClient) deleteUserInvite(ctx context.Context, inviteID string) error {
	path := fmt.Sprintf("%s/api/v2/user-invites/%s", m.baseURL().String(), url.PathEscape(inviteID))
	resp, err := m.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete user invite: %s (%d): %s", resp.Status, resp.StatusCode, string(body))
	}
	return nil
}

// suspendUser suspends the user by ID.
func (m *membershipAPIClient) suspendUser(ctx context.Context, userID string) error {
	path := fmt.Sprintf("%s/api/v2/users/%s/suspend", m.baseURL().String(), url.PathEscape(userID))
	resp, err := m.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("suspend user: %s (%d): %s", resp.Status, resp.StatusCode, string(body))
	}
	return nil
}

// restoreUser restores a suspended user by ID.
func (m *membershipAPIClient) restoreUser(ctx context.Context, userID string) error {
	path := fmt.Sprintf("%s/api/v2/users/%s/restore", m.baseURL().String(), url.PathEscape(userID))
	resp, err := m.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("restore user: %s (%d): %s", resp.Status, resp.StatusCode, string(body))
	}
	return nil
}

// deleteUser removes the user from the tailnet by ID.
func (m *membershipAPIClient) deleteUser(ctx context.Context, userID string) error {
	path := fmt.Sprintf("%s/api/v2/users/%s/delete", m.baseURL().String(), url.PathEscape(userID))
	resp, err := m.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete user: %s (%d): %s", resp.Status, resp.StatusCode, string(body))
	}
	return nil
}

// updateUserRole updates the user's role (API uses POST to /users/{id}/role).
func (m *membershipAPIClient) updateUserRole(ctx context.Context, userID, role string) error {
	path := fmt.Sprintf("%s/api/v2/users/%s/role", m.baseURL().String(), url.PathEscape(userID))
	body := map[string]string{"role": role}
	resp, err := m.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update user role: %s (%d): %s", resp.Status, resp.StatusCode, string(bodyBytes))
	}
	return nil
}
