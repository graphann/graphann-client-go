package graphann

import (
	"context"
	"net/url"
)

// settingsPath returns /v1/orgs/{orgID}/llm-settings.
func settingsPath(orgID string) string {
	return "/v1/orgs/" + url.PathEscape(orgID) + "/llm-settings"
}

// GetLLMSettings calls GET /v1/orgs/{orgID}/llm-settings.
func (c *Client) GetLLMSettings(ctx context.Context, orgID string) (*LLMSettings, error) {
	var out LLMSettings
	if err := c.do(ctx, "GET", settingsPath(orgID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateLLMSettings calls PATCH /v1/orgs/{orgID}/llm-settings. The server
// merges the supplied fields onto the current settings (partial update);
// the wrapped settings are unwrapped here so callers always receive a
// raw *LLMSettings on success.
func (c *Client) UpdateLLMSettings(ctx context.Context, orgID string, settings LLMSettings) (*LLMSettings, error) {
	var out UpdateLLMSettingsResponse
	if err := c.do(ctx, "PATCH", settingsPath(orgID), settings, &out, nil); err != nil {
		return nil, err
	}
	return &out.Settings, nil
}

// DeleteLLMSettings calls DELETE /v1/orgs/{orgID}/llm-settings.
func (c *Client) DeleteLLMSettings(ctx context.Context, orgID string) (*DeleteLLMSettingsResponse, error) {
	var out DeleteLLMSettingsResponse
	if err := c.do(ctx, "DELETE", settingsPath(orgID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
