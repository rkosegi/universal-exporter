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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/version"
	"github.com/rkosegi/universal-exporter/pkg/internal/cache"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/dom"
	"github.com/rkosegi/yaml-toolkit/fluent"
	"github.com/samber/lo"
)

var defConfig = &types.Config{
	Server: &types.ServerConfig{
		ListenAddress:  lo.ToPtr(types.DefaultListenAddress),
		HealthEndpoint: lo.ToPtr(types.DefaultHealthEndpoint),
		MetricsPath:    lo.ToPtr(types.DefaultMetricsEndpoint),
	},
	Vars: map[string]string{
		"Version": version.GetRevision(),
	},
	DefaultExporters: &types.DefaultExportersConfig{
		BuildInfo: lo.ToPtr(true),
	},
	Cache: &types.CacheConfig{
		TTL:      lo.ToPtr(types.DefaultCacheTTL),
		Capacity: lo.ToPtr(types.DefaultCacheCapacity),
		Instrumentation: &types.CacheInstrumentationConfig{
			Enabled: lo.ToPtr(true),
			Prefix:  lo.ToPtr(types.DefaultMetricPrefixHttpCache),
		},
	},
}

func initConfig(c *types.Config) (err error) {
	c = fluent.NewConfigHelper[types.Config]().Add(defConfig).Add(c).Result()
	c.GD = dom.Builder().Container()
	reg := prometheus.NewRegistry()
	c.GD.AddValueAt("prometheus.registry", dom.LeafNode(reg))

	if *c.DefaultExporters.BuildInfo {
		if err = reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
			return err
		}
	}
	if *c.DefaultExporters.Go {
		if err = reg.Register(collectors.NewGoCollector()); err != nil {
			return err
		}
	}
	if *c.DefaultExporters.BuildInfo {
		if err = reg.Register(collectors.NewBuildInfoCollector()); err != nil {
			return err
		}
	}

	for k, v := range c.MetricOpts {
		v.Name = k
		if !model.IsValidMetricName(model.LabelValue(v.Name)) {
			return fmt.Errorf("invalid metric name: %s", v.Name)
		}
		c.GD.AddValueAt(fmt.Sprintf("prometheus.opts.%s", k), dom.LeafNode(v))
	}
	c.GD.AddValueAt("http.cache", dom.LeafNode(cache.New(c.Cache, cache.WithPromRegistry(reg))))

	for _, v := range c.Targets {
		if v.Vars != nil {
			// c.GD.AddValueAt(fmt.Sprintf("targets.target_%d.vars", k), dom.DefaultNodeDecoderFn(v.Vars))
		}
	}

	return nil
}
