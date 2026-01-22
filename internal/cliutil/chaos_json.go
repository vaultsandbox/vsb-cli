package cliutil

import (
	"time"

	vaultsandbox "github.com/vaultsandbox/client-go"
)

// ChaosConfigJSONOptions controls which fields to include in chaos config JSON output.
type ChaosConfigJSONOptions struct {
	IncludeExpiry bool
}

// ChaosConfigJSON returns a map for JSON output with configurable fields.
func ChaosConfigJSON(cfg *vaultsandbox.ChaosConfig, opts ChaosConfigJSONOptions) map[string]interface{} {
	m := map[string]interface{}{
		"enabled": cfg.Enabled,
	}

	if opts.IncludeExpiry && cfg.ExpiresAt != nil {
		m["expiresAt"] = cfg.ExpiresAt.Format(time.RFC3339)
	}

	if cfg.Latency != nil && cfg.Latency.Enabled {
		m["latency"] = latencyToJSON(cfg.Latency)
	}
	if cfg.ConnectionDrop != nil && cfg.ConnectionDrop.Enabled {
		m["connectionDrop"] = connectionDropToJSON(cfg.ConnectionDrop)
	}
	if cfg.RandomError != nil && cfg.RandomError.Enabled {
		m["randomError"] = randomErrorToJSON(cfg.RandomError)
	}
	if cfg.Greylist != nil && cfg.Greylist.Enabled {
		m["greylist"] = greylistToJSON(cfg.Greylist)
	}
	if cfg.Blackhole != nil && cfg.Blackhole.Enabled {
		m["blackhole"] = blackholeToJSON(cfg.Blackhole)
	}

	return m
}

// ChaosConfigFullJSON returns a map for JSON output of full chaos config.
func ChaosConfigFullJSON(cfg *vaultsandbox.ChaosConfig) map[string]interface{} {
	return ChaosConfigJSON(cfg, ChaosConfigJSONOptions{IncludeExpiry: true})
}

// latencyToJSON converts latency config to JSON-friendly map.
func latencyToJSON(cfg *vaultsandbox.LatencyConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":     cfg.Enabled,
		"minDelayMs":  cfg.MinDelayMs,
		"maxDelayMs":  cfg.MaxDelayMs,
		"jitter":      cfg.Jitter,
		"probability": cfg.Probability,
	}
}

// connectionDropToJSON converts connection drop config to JSON-friendly map.
func connectionDropToJSON(cfg *vaultsandbox.ConnectionDropConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":     cfg.Enabled,
		"probability": cfg.Probability,
		"graceful":    cfg.Graceful,
	}
}

// randomErrorToJSON converts random error config to JSON-friendly map.
func randomErrorToJSON(cfg *vaultsandbox.RandomErrorConfig) map[string]interface{} {
	errorTypes := make([]string, len(cfg.ErrorTypes))
	for i, t := range cfg.ErrorTypes {
		errorTypes[i] = string(t)
	}
	return map[string]interface{}{
		"enabled":    cfg.Enabled,
		"errorRate":  cfg.ErrorRate,
		"errorTypes": errorTypes,
	}
}

// greylistToJSON converts greylist config to JSON-friendly map.
func greylistToJSON(cfg *vaultsandbox.GreylistConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":       cfg.Enabled,
		"retryWindowMs": cfg.RetryWindowMs,
		"maxAttempts":   cfg.MaxAttempts,
		"trackBy":       string(cfg.TrackBy),
	}
}

// blackholeToJSON converts blackhole config to JSON-friendly map.
func blackholeToJSON(cfg *vaultsandbox.BlackholeConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":         cfg.Enabled,
		"triggerWebhooks": cfg.TriggerWebhooks,
	}
}

// ChaosEnabledTypes returns a list of enabled chaos type names.
func ChaosEnabledTypes(cfg *vaultsandbox.ChaosConfig) []string {
	var types []string
	if cfg.Latency != nil && cfg.Latency.Enabled {
		types = append(types, "latency")
	}
	if cfg.ConnectionDrop != nil && cfg.ConnectionDrop.Enabled {
		types = append(types, "connection-drop")
	}
	if cfg.RandomError != nil && cfg.RandomError.Enabled {
		types = append(types, "random-error")
	}
	if cfg.Greylist != nil && cfg.Greylist.Enabled {
		types = append(types, "greylist")
	}
	if cfg.Blackhole != nil && cfg.Blackhole.Enabled {
		types = append(types, "blackhole")
	}
	return types
}

// ChaosSummaryJSON returns a summary of chaos config for inbox info output.
func ChaosSummaryJSON(cfg *vaultsandbox.ChaosConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled": cfg.Enabled,
		"types":   ChaosEnabledTypes(cfg),
	}
}
