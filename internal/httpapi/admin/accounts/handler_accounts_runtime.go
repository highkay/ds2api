package accounts

import (
	"slices"

	"ds2api/internal/config"
)

func runtimeProbeResponseMap(probe config.AccountRuntimeProbe) map[string]any {
	return map[string]any{
		"token_valid":       boolPtrValue(probe.TokenValid),
		"token_http_status": probe.TokenHTTPStatus,
		"token_code":        probe.TokenCode,
		"token_biz_code":    probe.TokenBizCode,
		"token_message":     probe.TokenMessage,
		"capabilities":      capabilityProbeResponseMap(probe.Capabilities),
		"capability_error":  probe.CapabilityError,
		"checked_at":        probe.CheckedAt,
	}
}

func tokenStatusResponseMap(probe config.AccountRuntimeProbe) map[string]any {
	return map[string]any{
		"valid":       boolPtrValue(probe.TokenValid),
		"http_status": probe.TokenHTTPStatus,
		"code":        probe.TokenCode,
		"biz_code":    probe.TokenBizCode,
		"message":     probe.TokenMessage,
		"checked_at":  probe.CheckedAt,
	}
}

func capabilityProbeResponseMap(cap config.AccountCapabilityProbe) map[string]any {
	return map[string]any{
		"vision":     boolPtrValue(cap.Vision),
		"models":     slices.Clone(cap.Models),
		"checked_at": cap.CheckedAt,
		"source":     cap.Source,
	}
}

func runtimeProbeHasData(probe config.AccountRuntimeProbe) bool {
	return probe.TokenValid != nil ||
		probe.CheckedAt != 0 ||
		probe.Capabilities.Vision != nil ||
		len(probe.Capabilities.Models) > 0 ||
		probe.Capabilities.CheckedAt != 0 ||
		probe.CapabilityError != ""
}

func boolPtrValue(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func boolPtr(v bool) *bool {
	return &v
}
