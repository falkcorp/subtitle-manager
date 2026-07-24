// file: pkg/cache/proto.go
// version: 1.1.1
// guid: 94a9a33b-5e6e-4b6c-9c34-24e2f5a1e3a1

package cache

import (
	commonpb "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	"google.golang.org/protobuf/types/known/durationpb"
)

// TTLConfigToProto converts a TTLConfig to a map of CachePolicy messages
// keyed by configuration name.
func TTLConfigToProto(cfg TTLConfig) map[string]*commonpb.CachePolicy {
	policies := make(map[string]*commonpb.CachePolicy)

	providerPolicy := &commonpb.CachePolicy{}
	providerPolicy.SetDefaultTtl(durationpb.New(cfg.ProviderSearchResults))
	policies["provider_search_results"] = providerPolicy

	searchPolicy := &commonpb.CachePolicy{}
	searchPolicy.SetDefaultTtl(durationpb.New(cfg.SearchResults))
	policies["search_results"] = searchPolicy

	tmdbPolicy := &commonpb.CachePolicy{}
	tmdbPolicy.SetDefaultTtl(durationpb.New(cfg.TMDBMetadata))
	policies["tmdb_metadata"] = tmdbPolicy

	translationPolicy := &commonpb.CachePolicy{}
	translationPolicy.SetDefaultTtl(durationpb.New(cfg.TranslationResults))
	policies["translation_results"] = translationPolicy

	sessionPolicy := &commonpb.CachePolicy{}
	sessionPolicy.SetDefaultTtl(durationpb.New(cfg.UserSessions))
	policies["user_sessions"] = sessionPolicy

	apiPolicy := &commonpb.CachePolicy{}
	apiPolicy.SetDefaultTtl(durationpb.New(cfg.APIResponses))
	policies["api_responses"] = apiPolicy

	return policies
}

// TTLConfigFromProto converts a map of CachePolicy messages to TTLConfig.
// Unknown keys are ignored.
func TTLConfigFromProto(policies map[string]*commonpb.CachePolicy) TTLConfig {
	var cfg TTLConfig
	if p, ok := policies["provider_search_results"]; ok && p.GetDefaultTtl() != nil {
		cfg.ProviderSearchResults = p.GetDefaultTtl().AsDuration()
	}
	if p, ok := policies["search_results"]; ok && p.GetDefaultTtl() != nil {
		cfg.SearchResults = p.GetDefaultTtl().AsDuration()
	}
	if p, ok := policies["tmdb_metadata"]; ok && p.GetDefaultTtl() != nil {
		cfg.TMDBMetadata = p.GetDefaultTtl().AsDuration()
	}
	if p, ok := policies["translation_results"]; ok && p.GetDefaultTtl() != nil {
		cfg.TranslationResults = p.GetDefaultTtl().AsDuration()
	}
	if p, ok := policies["user_sessions"]; ok && p.GetDefaultTtl() != nil {
		cfg.UserSessions = p.GetDefaultTtl().AsDuration()
	}
	if p, ok := policies["api_responses"]; ok && p.GetDefaultTtl() != nil {
		cfg.APIResponses = p.GetDefaultTtl().AsDuration()
	}
	return cfg
}
