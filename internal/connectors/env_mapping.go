// Package connectors provides connector utilities including encryption and environment variable mapping.
package connectors

import (
	"encoding/json"
	"strings"
)

// ConnectorEnvVarMap maps connector slugs to their corresponding environment variable names.
// When a connector is enabled for a session, its API key is injected as this env var.
var ConnectorEnvVarMap = map[string]string{
	"anthropic":   "ANTHROPIC_API_KEY",
	"openai":      "OPENAI_API_KEY",
	"google_ai":   "GOOGLE_AI_API_KEY",
	"deepseek":    "DEEPSEEK_API_KEY",
	"mistral":     "MISTRAL_API_KEY",
	"openrouter":  "OPENROUTER_API_KEY",
	"huggingface": "HF_TOKEN",
	"github":      "GITHUB_TOKEN",
	"gitea":       "GITEA_TOKEN",
	"linear":      "LINEAR_API_KEY",
	"slack":       "SLACK_TOKEN",
	"discord":     "DISCORD_TOKEN",
	"sendgrid":    "SENDGRID_API_KEY",
	"webhook":     "WEBHOOK_SECRET",
	"vercel":      "VERCEL_TOKEN",
	"supabase":    "SUPABASE_KEY",
	"stripe":      "STRIPE_SECRET_KEY",
	"sentry":      "SENTRY_AUTH_TOKEN",
	"n0":          "N0_API_KEY",
}

// connectorLegacyEnvVarMap maps connector slugs to legacy environment variable
// names that are still injected for backward compatibility (in addition to the
// primary env var from ConnectorEnvVarMap).
var connectorLegacyEnvVarMap = map[string][]string{
	// n0 was previously named "nzero"; keep the old env var alive for
	// existing agent prompts and scripts.
	"n0": {"NZERO_API_KEY"},
}

// ConnectorExtraEnvVarMap maps connector slugs to extra environment variables
// that should be extracted from extra_config JSON and injected into the session.
// The map value is: JSON key → env var names (first is primary, rest are legacy aliases).
var ConnectorExtraEnvVarMap = map[string]map[string][]string{
	"n0": {
		"domain": {"N0_DOMAIN", "NZERO_DOMAIN"},
	},
}

// GetLegacyEnvVarNames returns legacy environment variable names for a connector
// slug that should also receive the API key for backward compatibility.
func GetLegacyEnvVarNames(slug string) []string {
	return connectorLegacyEnvVarMap[slug]
}

// GetEnvVarName returns the environment variable name for a connector slug.
// Falls back to UPPERCASE_SLUG_API_KEY if the slug is not in the predefined map.
func GetEnvVarName(slug string) string {
	if name, ok := ConnectorEnvVarMap[slug]; ok {
		return name
	}
	// Fallback: uppercase slug + _API_KEY
	return strings.ToUpper(strings.ReplaceAll(slug, "-", "_")) + "_API_KEY"
}

// GetExtraEnvVars extracts additional environment variables from a connector's extra_config JSON.
// Returns a map of env var name → value for any configured extra vars.
func GetExtraEnvVars(slug string, extraConfigJSON string) map[string]string {
	result := make(map[string]string)

	mapping, ok := ConnectorExtraEnvVarMap[slug]
	if !ok || extraConfigJSON == "" {
		return result
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(extraConfigJSON), &config); err != nil {
		return result
	}

	for jsonKey, envVars := range mapping {
		if val, exists := config[jsonKey]; exists {
			if strVal, ok := val.(string); ok && strVal != "" {
				for _, envVar := range envVars {
					result[envVar] = strVal
				}
			}
		}
	}

	return result
}
