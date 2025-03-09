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

package cache

import (
	"net/url"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/fluent"
	"github.com/samber/lo"
)

var (
	NoopCache types.HttpResponseCache = noopCache{}
	defCfg                            = &types.CacheConfig{
		TTL:      lo.ToPtr(types.DefaultCacheTTL),
		Capacity: lo.ToPtr(types.DefaultCacheCapacity),
		Instrumentation: &types.CacheInstrumentationConfig{
			Enabled: lo.ToPtr(true),
			Prefix:  lo.ToPtr(types.DefaultMetricPrefixHttpCache),
		},
	}
)

type noopCache struct{}

func (n noopCache) Put(string, *types.ParsedHttpResponse) {}
func (n noopCache) Get(string) *types.ParsedHttpResponse  { return nil }

type noopCollector struct{}

func (n noopCollector) Describe(chan<- *prometheus.Desc) {}
func (n noopCollector) Collect(chan<- prometheus.Metric) {}

type noopRegisterer struct{}

func (n noopRegisterer) Register(prometheus.Collector) error  { return nil }
func (n noopRegisterer) MustRegister(...prometheus.Collector) {}
func (n noopRegisterer) Unregister(prometheus.Collector) bool { return false }

type Opt func(*impl)

func WithPromRegistry(reg prometheus.Registerer) Opt {
	return func(i *impl) {
		i.r = reg
	}
}

type impl struct {
	c           *ttlcache.Cache[string, *types.ParsedHttpResponse]
	col         prometheus.Collector
	r           prometheus.Registerer
	hitCounter  *prometheus.CounterVec
	missCounter *prometheus.CounterVec
}

func key2host(key string) string {
	if u, err := url.Parse(key); err == nil {
		return u.Host
	}
	return key
}

func (h *impl) Describe(ch chan<- *prometheus.Desc) {
	h.col.Describe(ch)
}

func (h *impl) Collect(ch chan<- prometheus.Metric) {
	h.col.Collect(ch)
}

func (h *impl) Put(key string, value *types.ParsedHttpResponse) {
	h.c.Set(key, value, ttlcache.DefaultTTL)
}

func (h *impl) onCacheMiss(key string) {
	if h.missCounter != nil {
		h.missCounter.WithLabelValues(key2host(key)).Inc()
	}
}

func (h *impl) onCacheHit(key string) {
	if h.hitCounter != nil {
		h.hitCounter.WithLabelValues(key2host(key)).Inc()
	}
}

func (h *impl) Get(key string) *types.ParsedHttpResponse {
	i := h.c.Get(key)
	if i == nil {
		h.onCacheMiss(key)
		return nil
	}
	h.onCacheHit(key)
	return i.Value()
}

// New instantiates ttlcache.Cache from configuration
func New(cfg *types.CacheConfig, opts ...Opt) types.HttpResponseCache {
	cfg = fluent.NewConfigHelper[types.CacheConfig]().Add(defCfg).Add(cfg).Result()
	c := ttlcache.New[string, *types.ParsedHttpResponse](
		ttlcache.WithDisableTouchOnHit[string, *types.ParsedHttpResponse](),
		ttlcache.WithTTL[string, *types.ParsedHttpResponse](*cfg.TTL),
		ttlcache.WithCapacity[string, *types.ParsedHttpResponse](uint64(*cfg.Capacity)),
	)
	go c.Start()

	i := &impl{
		c:   c,
		r:   noopRegisterer{},
		col: noopCollector{},
	}
	for _, opt := range opts {
		opt(i)
	}

	if cfg.Instrumentation != nil && cfg.Instrumentation.Enabled != nil && *cfg.Instrumentation.Enabled {
		if cfg.Instrumentation.Prefix == nil {
			cfg.Instrumentation.Prefix = lo.ToPtr(types.DefaultMetricPrefixHttpCache)
		}
		i.hitCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: *cfg.Instrumentation.Prefix + "_hit",
			Help: "HTTP response cache hit count",
		}, []string{"host"})
		i.missCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: *cfg.Instrumentation.Prefix + "_miss",
			Help: "HTTP response cache miss count",
		}, []string{"host"})
		i.col = i
		_ = i.r.Register(i)
	}

	return i
}
