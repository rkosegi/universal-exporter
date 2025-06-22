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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/yaml-pipeline/pkg/pipeline"
)

type HttpClientService interface {
	pipeline.Service
	prometheus.Collector
	RoundTripper() http.RoundTripper
	Start() error
}

// MetricService provides access to configured metrics
type MetricService interface {
	pipeline.Service
	prometheus.Collector
	GetRef(ref string) (*MetricOptsSpec, error)
	Start() error
}
