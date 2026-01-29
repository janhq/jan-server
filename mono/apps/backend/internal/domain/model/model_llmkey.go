package model

import (
	"regexp"
	"strings"
)

// NormalizeModelKey returns a canonical "<vendor>/<model>" key.
// It tries to infer the underlying vendor from the raw name and provider kind.
//
// Examples:
//
//	NormalizeModelKey(ProviderOpenRouter, "anthropic/claude-3.5-sonnet") => "anthropic/claude-3.5-sonnet"
//	NormalizeModelKey(ProviderAWSBedrock, "anthropic.claude-3-5-sonnet-20240620-v1:0") => "anthropic/claude-3-5-sonnet-20240620-v1"
//	NormalizeModelKey(ProviderOllama, "llama3:8b-instruct") => "meta/llama3-8b-instruct"
//	NormalizeModelKey(ProviderVercelAI, "openai:gpt-4o-mini") => "openai/gpt-4o-mini"
//	NormalizeModelKey(ProviderGoogle, "models/gemini-1.5-flash-001") => "google/gemini-1.5-flash-001"
//	NormalizeModelKey(ProviderGroq, "mixtral-8x7b-32768") => "mistral/mixtral-8x7b-32768"
//
// Additional test cases:
//
//	# Aggregator with unknown owner - falls back to provider vendor
//	NormalizeModelKey(ProviderJan, "aibrix/jan-v1-4b") => "jan/jan-v1-4b"
//	NormalizeModelKey(ProviderJan, "jan-v1-4b") => "jan/jan-v1-4b"
//
//	# Aggregator with known vendor - preserves vendor
//	NormalizeModelKey(ProviderOpenRouter, "meta-llama/Llama-3-8B-Instruct") => "meta/llama-3-8b-instruct"
//	NormalizeModelKey(ProviderReplicate, "anthropic/claude-3.5-sonnet") => "anthropic/claude-3.5-sonnet"
//
//	# Version handling
//	NormalizeModelKey(ProviderReplicate, "owner/model:v1.0") => "vendor/model-v1.0"
//
//	# Family inference
//	NormalizeModelKey(ProviderGroq, "llama-3.1-70b-versatile") => "meta/llama-3.1-70b-versatile"
//	NormalizeModelKey(ProviderCustom, "mixtral-8x7b") => "mistral/mixtral-8x7b"
//
//	# Special prefixes
//	NormalizeModelKey(ProviderGoogle, "models/gemini-pro") => "google/gemini-pro"
//	NormalizeModelKey(ProviderAWSBedrock, "meta.llama3-70b-instruct-v1:0") => "meta/llama3-70b-instruct-v1"
//
//	# Colon-separated vendor:model pattern
//	NormalizeModelKey(ProviderVercelAI, "anthropic:claude-3-opus") => "anthropic/claude-3-opus"
//	NormalizeModelKey(ProviderOllama, "qwen2:7b") => "qwen/qwen2-7b"
func NormalizeModelKey(pk ProviderKind, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// 1) Try cross-provider patterns first (highest priority)

	// Pattern: "vendor:model" where vendor is a known vendor
	if v, m, ok := splitPair(raw, ":"); ok && likelyVendor(v) {
		return joinKM(slug(v), slug(m))
	}

	// Pattern: Google's "models/model-name" format
	if pk == ProviderGoogle && strings.HasPrefix(strings.ToLower(raw), "models/") {
		name := strings.TrimPrefix(raw, "models/")
		return joinKM("google", slug(name))
	}

	// Pattern: AWS Bedrock "vendor.model-name:version" format
	if pk == ProviderAWSBedrock {
		return normalizeBedrockModel(raw)
	}

	// Pattern: "owner/model[:version]" - common for aggregators and repos
	// For aggregators, use provider as fallback vendor for unknown owners
	fallbackVendor := ""
	switch pk {
	case ProviderJan, ProviderOpenRouter, ProviderTogetherAI, ProviderVercelAI, ProviderDeepInfra, ProviderReplicate, ProviderHuggingFace:
		fallbackVendor = getProviderVendorName(pk)
	}
	if vendor, model, ok := parseOwnerModelPair(raw, fallbackVendor); ok {
		return joinKM(vendor, model)
	}

	// Pattern: Ollama "family:tag" format
	if pk == ProviderOllama && strings.Contains(raw, ":") {
		return normalizeOllamaModel(raw)
	}

	// 2) Provider-specific defaults (no owner prefix detected)
	return normalizeByProvider(pk, raw)
}

// normalizeBedrockModel handles AWS Bedrock's "vendor.model:version" format.
func normalizeBedrockModel(raw string) string {
	main := strings.SplitN(raw, ":", 2)[0] // drop ":0" etc
	if segs := strings.SplitN(main, ".", 2); len(segs) == 2 {
		vendor := slug(segs[0])
		model := slug(segs[1])
		vendor = remapVendorFromFamily(vendor, model)
		return joinKM(vendor, model)
	}
	// fallback: infer from family
	return inferFromFamily(raw)
}

// normalizeOllamaModel handles Ollama's "family:tag" format.
func normalizeOllamaModel(raw string) string {
	base, tag, _ := splitPair(raw, ":")
	model := slug(base + "-" + tag)
	vendor := vendorFromFamilyPrefix(base)
	return joinKM(vendor, model)
}

// normalizeByProvider applies provider-specific normalization rules.
func normalizeByProvider(pk ProviderKind, raw string) string {
	switch pk {
	case ProviderOpenAI, ProviderAzureOpenAI:
		return joinKM("openai", slug(raw))
	case ProviderAnthropic:
		return joinKM("anthropic", slug(raw))
	case ProviderGoogle:
		return joinKM("google", slug(stripModelsPrefix(raw)))
	case ProviderMistral:
		return joinKM("mistral", slug(raw))
	case ProviderCohere:
		return joinKM("cohere", slug(raw))
	case ProviderGroq:
		// Groq hosts many families; infer vendor by family prefix
		return inferFromFamily(raw)
	case ProviderPerplexity:
		// Prefer perplexity for their own models (pplx/sonar)
		r := strings.ToLower(raw)
		if strings.HasPrefix(r, "pplx") || strings.Contains(r, "sonar") {
			return joinKM("perplexity", slug(raw))
		}
		return inferFromFamily(raw)
	case ProviderJan, ProviderOpenRouter, ProviderTogetherAI, ProviderVercelAI, ProviderDeepInfra, ProviderReplicate, ProviderHuggingFace:
		// Aggregators: use provider name as vendor for bare model names
		return joinKM(getProviderVendorName(pk), slug(raw))
	case ProviderCustom:
		// Custom providers: best effort family inference
		return inferFromFamily(raw)
	default:
		return inferFromFamily(raw)
	}
}

// ---- helpers ----

// knownVendors is the single source of truth for all recognized AI model vendors.
// This prevents duplication and makes it easier to add new vendors.
var knownVendors = map[string]bool{
	"openai":     true,
	"anthropic":  true,
	"gemini":     true,
	"google":     true,
	"mistral":    true,
	"mistralai":  true,
	"meta":       true,
	"meta-llama": true,
	"cohere":     true,
	"qwen":       true,
	"qwen2":      true,
	"qwen2.5":    true,
	"qwen3":      true,
	"tii":        true,
	"tiiuae":     true,
	"databricks": true,
	"aws":        true,
	"azure":      true,
	"perplexity": true,
	"microsoft":  true,
	"deepmind":   true,
	"01-ai":      true,
	"zhipu":      true,
}

// parseOwnerModelPair extracts and normalizes owner/model from "owner/model[:version]" format.
// Returns empty strings if the format doesn't match or if the owner contains spaces.
// If vendor is unrecognized and fallbackVendor is provided, uses fallback instead.
func parseOwnerModelPair(raw string, fallbackVendor string) (vendor, model string, ok bool) {
	owner, name, found := splitPair(raw, "/")
	if !found || owner == "" || name == "" || strings.Contains(owner, " ") {
		return "", "", false
	}

	ownerSlug := slug(owner)
	// Handle version tags: "name:version" -> "name-version"
	if n, ver, has := splitPair(name, ":"); has && n != "" {
		name = n + "-" + ver
	}
	modelSlug := slug(strings.ReplaceAll(name, ":", "-"))

	vendor = remapVendorFromFamily(ownerSlug, modelSlug)
	// If vendor is unrecognized, use fallback
	if vendor == ownerSlug && !isKnownVendor(vendor) && fallbackVendor != "" {
		vendor = fallbackVendor
	}

	return vendor, modelSlug, true
}

var nonAlnumDashDot = regexp.MustCompile(`[^a-z0-9\-\.:\/]`)

func slug(input string) string {
	output := strings.TrimSpace(strings.ToLower(input))
	output = strings.ReplaceAll(output, "_", "-")
	output = strings.Join(strings.Fields(output), "-")
	output = nonAlnumDashDot.ReplaceAllString(output, "")
	// collapse "meta-llama" common HF owner; keep dots/colons until we normalize them out above
	return output
}

func joinKM(vendor, model string) string {
	vendor = strings.Trim(vendor, "/")
	model = strings.Trim(model, "/")
	if vendor == "" {
		vendor = "unknown"
	}
	return vendor + "/" + model
}

func splitPair(s, sep string) (string, string, bool) {
	i := strings.Index(s, sep)
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// likelyVendor checks if a string is likely a vendor name that appears as a prefix.
// This is a subset of known vendors commonly used in "vendor:model" patterns.
func likelyVendor(s string) bool {
	v := strings.ToLower(s)
	// Common vendors that appear as prefixes in aggregator APIs
	switch v {
	case "openai", "anthropic", "gemini", "google", "mistral", "meta", "cohere":
		return true
	default:
		return isKnownVendor(v)
	}
}

// isKnownVendor checks if a string matches any recognized AI model vendor.
func isKnownVendor(s string) bool {
	return knownVendors[strings.ToLower(s)]
}

func getProviderVendorName(pk ProviderKind) string {
	switch pk {
	case ProviderJan:
		return "jan"
	case ProviderOpenRouter:
		return "openrouter"
	case ProviderTogetherAI:
		return "together"
	case ProviderVercelAI:
		return "vercel"
	case ProviderDeepInfra:
		return "deepinfra"
	case ProviderReplicate:
		return "replicate"
	case ProviderHuggingFace:
		return "huggingface"
	case ProviderZImage:
		return "zimage"
	default:
		return strings.ToLower(string(pk))
	}
}

func stripModelsPrefix(s string) string {
	if strings.HasPrefix(strings.ToLower(s), "models/") {
		return s[7:]
	}
	return s
}

// Maps family hints to real vendors (e.g., "llama3" -> "meta")
func vendorFromFamilyPrefix(modelBase string) string {
	m := strings.ToLower(modelBase)
	switch {
	case strings.HasPrefix(m, "llama"):
		return "meta"
	case strings.HasPrefix(m, "gemma"):
		return "google"
	case strings.HasPrefix(m, "mixtral"), strings.HasPrefix(m, "mistral"):
		return "mistral"
	case strings.HasPrefix(m, "qwen"):
		return "qwen"
	case strings.HasPrefix(m, "phi"):
		return "microsoft"
	case strings.HasPrefix(m, "yi"):
		return "01-ai"
	case strings.HasPrefix(m, "glm"), strings.HasPrefix(m, "chatglm"):
		return "zhipu"
	default:
		return "unknown"
	}
}

// If owner looks like a family (meta-llama) prefer the brand vendor; else use owner.
func remapVendorFromFamily(owner, model string) string {
	switch owner {
	case "meta-llama", "meta":
		return "meta"
	case "google", "deepmind":
		return "google"
	case "mistralai", "mistral":
		return "mistral"
	case "anthropic":
		return "anthropic"
	case "qwen", "qwen2", "qwen2.5", "qwen3":
		return "qwen"
	case "tiiuae", "tii":
		return "tii"
	case "openai":
		return "openai"
	case "cohere":
		return "cohere"
	default:
		// Try infer from model family prefix
		v := vendorFromFamilyPrefix(model)
		if v != "unknown" {
			return v
		}
		return owner
	}
}

// As a last resort, infer vendor from recognizable family in a bare model name.
func inferFromFamily(raw string) string {
	r := strings.ToLower(raw)
	// Handle name:tag patterns (ollama-like)
	if base, tag, ok := splitPair(r, ":"); ok {
		r = base + "-" + tag
	}
	r = stripModelsPrefix(r)
	model := slug(strings.ReplaceAll(r, "/", "-"))
	// Try to extract first token as family base
	family := model
	if i := strings.IndexAny(model, "-_."); i > 0 {
		family = model[:i]
	}
	vendor := vendorFromFamilyPrefix(family)
	// Special cases
	if strings.Contains(model, "claude") {
		vendor = "anthropic"
	}
	if strings.Contains(model, "gpt") || strings.Contains(model, "o1") {
		vendor = "openai"
	}
	if strings.HasPrefix(model, "gemini") || strings.HasPrefix(model, "google") {
		vendor = "google"
	}
	return joinKM(vendor, model)
}
