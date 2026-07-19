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
)

func DefaultConfig() *types.Config {
	return &types.Config{
		HttpClient: &types.HttpClientServiceConfig{
			Timeout: new(time.Second * 15),
			Instrumentation: &types.InstrumentationConfigFragment{
				Enabled: new(true),
				Prefix:  new(types.PromNamespace),
			},
			Cache: &types.CacheConfig{
				Enabled:  new(true),
				TTL:      new(types.DefaultCacheTTL),
				Capacity: new(types.DefaultCacheCapacity),
				Instrumentation: &types.InstrumentationConfigFragment{
					Enabled: new(true),
					Prefix:  new(types.DefaultMetricPrefixHttpCache),
				},
			},
		},
		Server: &types.ServerConfig{
			HealthEndpoint: new(types.DefaultHealthEndpoint),
			MetricsPath:    new(types.DefaultMetricsEndpoint),
		},
		Vars: map[string]string{
			"Version": version.GetRevision(),
		},
		DefaultExporters: &types.DefaultExportersConfig{
			BuildInfo:             new(true),
			Version:               new(true),
			Process:               new(true),
			Go:                    new(true),
			InstrumentHttpHandler: new(true),
		},
	}
}
