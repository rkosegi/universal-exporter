/*
Copyright 2025 Richard Kosegi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"time"

	"github.com/prometheus/common/version"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/samber/lo"
)

func DefaultConfig() *types.Config {
	return &types.Config{
		HttpClient: &types.HttpClientServiceConfig{
			Timeout: lo.ToPtr(time.Second * 15),
			Instrumentation: &types.InstrumentationConfigFragment{
				Enabled: lo.ToPtr(true),
				Prefix:  lo.ToPtr(types.PromNamespace),
			},
			Cache: &types.CacheConfig{
				TTL:      lo.ToPtr(types.DefaultCacheTTL),
				Capacity: lo.ToPtr(types.DefaultCacheCapacity),
				Instrumentation: &types.InstrumentationConfigFragment{
					Enabled: lo.ToPtr(true),
					Prefix:  lo.ToPtr(types.DefaultMetricPrefixHttpCache),
				},
			},
		},
		Server: &types.ServerConfig{
			HealthEndpoint: lo.ToPtr(types.DefaultHealthEndpoint),
			MetricsPath:    lo.ToPtr(types.DefaultMetricsEndpoint),
		},
		Vars: map[string]string{
			"Version": version.GetRevision(),
		},
		DefaultExporters: &types.DefaultExportersConfig{
			BuildInfo:             lo.ToPtr(true),
			Version:               lo.ToPtr(true),
			Process:               lo.ToPtr(true),
			Go:                    lo.ToPtr(true),
			InstrumentHttpHandler: lo.ToPtr(true),
		},
	}
}
