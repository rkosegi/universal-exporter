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

package types

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/yaml-toolkit/dom"
	"github.com/rkosegi/yaml-toolkit/pipeline"
)

// MetricOptsSpec is configuration that includes prometheus.Opts and label names
type MetricOptsSpec struct {
	// Name is fully qualified metric name
	Name string
	// Help is info about metric
	Help string
	// LabelsNames are label names that must be populated when setting metric value
	LabelsNames []string `json:"labelsNames" yaml:"labelsNames"`
	// ConstLabels are fixed labels with their values that will be attached to metric
	ConstLabels map[string]string `json:"constLabels" yaml:"constLabels"`
}

func (p *MetricOptsSpec) AsOpts() prometheus.Opts {
	return prometheus.Opts{
		Name:        p.Name,
		Help:        p.Help,
		ConstLabels: p.ConstLabels,
	}
}

type HttpFetchConfig struct {
	Url     string
	Headers map[string]string
}

// ScrapeTarget defines how target is being scraped
type ScrapeTarget struct {
	// Vars are target-specific variables that will be merged with global ones before they are used.
	Vars map[string]string

	HttpFetch *HttpFetchConfig `json:"httpFetch" yaml:"httpFetch"`

	// Steps are actual pipeline steps
	Steps pipeline.ChildActions `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type HttpClientServiceConfig struct {
	Timeout *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Cache   *CacheConfig   `json:"cache" yaml:"cache"`
}

type CacheInstrumentationConfig struct {
	Enabled *bool   `json:"enabled" yaml:"enabled"`
	Prefix  *string `json:"prefix" yaml:"prefix"`
}

// CacheConfig is used to configure TTL cache for HTTP responses.
type CacheConfig struct {
	TTL *time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	// Capacity determines max. number of items in cache
	Capacity *int `json:"capacity,omitempty" yaml:"capacity,omitempty"`

	// Instrumentation enables cache instrumentation
	Instrumentation *CacheInstrumentationConfig `json:"instrumentation,omitempty" yaml:"instrumentation,omitempty"`
}

// DefaultExportersConfig configuration of default exporters
type DefaultExportersConfig struct {
	// BuildInfo flag to enable collectors.NewBuildInfoCollector()
	BuildInfo *bool `json:"buildInfo,omitempty" yaml:"buildInfo,omitempty"`
	// Go flag to enable collectors.NewGoCollector()
	Go *bool `json:"go,omitempty" yaml:"go,omitempty"`
	// Process flag to enable collectors.NewProcessCollector()
	Process *bool `json:"process,omitempty" yaml:"process,omitempty"`
	// InstrumentHttpHandler flag to wrap http handler with promhttp.InstrumentMetricHandler
	InstrumentHttpHandler *bool `json:"instrumentHttpHandler,omitempty" yaml:"instrumentHttpHandler,omitempty"`
}

type ServerConfig struct {
	// ListenAddress is address to listen on, default :9999
	ListenAddress *string `json:"listenAddress,omitempty" yaml:"listenAddress,omitempty"`
	// HealthEndpoint HTTP route for health check. Default value is /healthz
	HealthEndpoint *string `json:"healthEndpoint,omitempty" yaml:"healthEndpoint,omitempty"`
	// MetricsPath HTTP route for handling metrics. Default value is /metrics
	MetricsPath *string `json:"metricsPath,omitempty" yaml:"metricsPath,omitempty"`
}

type Config struct {
	// Server is server configuration
	Server *ServerConfig `json:"server,omitempty" yaml:"server,omitempty"`

	// Vars is map of global variables
	Vars map[string]string `json:"vars" yaml:"vars"`

	// MetricOpts is map of names to MetricOptsSpec.
	// These are referred to by pipeline functions during transformation.
	MetricOpts map[string]MetricOptsSpec `json:"metricOpts" yaml:"metricOpts"`

	// Whether to register built-in descriptors such as Go GC, process etc.
	DefaultExporters *DefaultExportersConfig `json:"defaultExporters,omitempty" yaml:"defaultExporters,omitempty"`

	// Targets to scrape. REQUIRED.
	Targets map[string]ScrapeTarget `json:"targets" yaml:"targets"`

	// Global data tree. This is where internal state is stored. Do not touch.
	GD dom.ContainerBuilder `json:"-" yaml:"-"`
}

// misc structs

type ParsedHttpResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type GenericCache[K string, V any] interface {
	// Put puts V into cache
	Put(key K, value *V)
	// Get retrieves V from cache. If no entry exist with key K, then nil is returned.
	Get(key K) *V
}

// HttpResponseCache is TTL based cache for HTTP responses
// Key to this cache is just a URL, it makes sense to cache only GET responses anyway.
type HttpResponseCache GenericCache[string, ParsedHttpResponse]

type NamedLookup[T any] interface {
	Get(name string) (T, bool)
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type HttpClientService interface {
	GenericCache[string, ParsedHttpResponse]
	Config() HttpClientServiceConfig
	Client() http.Client
	Doer() HTTPDoer
}
