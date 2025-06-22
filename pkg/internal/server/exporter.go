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
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/internal/ops"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-pipeline/pkg/pipeline"
	"github.com/rkosegi/yaml-toolkit/dom"
)

type pipelineLogAdapter struct {
	l  *slog.Logger
	rl int
}

func (p *pipelineLogAdapter) OnBefore(ctx pipeline.ActionContext) {
	p.l.Debug("pipeline event", "event_type", "OnBefore", "recursion_level", p.rl, "action", ctx.Action().String())
	p.rl++
}

func (p *pipelineLogAdapter) OnAfter(ctx pipeline.ActionContext, err error) {
	p.rl--
	if err != nil {
		p.l.Error("pipeline error", "recursion_level", p.rl, "action", ctx.Action().String(), "error", err)
	} else {
		p.l.Debug("pipeline event", "event_type", "OnAfter", "recursion_level", p.rl, "action", ctx.Action().String())
	}
}

func (p *pipelineLogAdapter) OnLog(ctx pipeline.ActionContext, v ...interface{}) {
	p.l.Debug("pipeline event", "event_type", "OnLog", "recursion_level", p.rl, "action", ctx.Action(), "log_args", v)
}

type pipelineCollector struct {
	gc             *types.Config
	l              *slog.Logger
	ms             types.MetricService
	lastErr        prometheus.Gauge
	up             prometheus.Gauge
	scrapeFailures *prometheus.CounterVec
	scrapeSum      *prometheus.SummaryVec
	hcs            types.HttpClientService
}

func (p *pipelineCollector) Describe(ch chan<- *prometheus.Desc) {
	p.up.Describe(ch)
	p.ms.Describe(ch)
	p.hcs.Describe(ch)
}

func applyVars(kvs map[string]string, gd dom.ContainerBuilder) {
	for k, v := range kvs {
		gd.AddValueAt("vars."+k, dom.LeafNode(v))
	}
}

func (p *pipelineCollector) Collect(ch chan<- prometheus.Metric) {
	p.up.Set(1)
	gd := dom.Builder().Container()
	ex := pipeline.New(
		pipeline.WithData(gd),
		pipeline.WithListener(&pipelineLogAdapter{
			l: p.l.With("component", "pipeline"),
		}),
		pipeline.WithServices(map[string]pipeline.Service{
			"HttpClient":    p.hcs,
			"MetricService": p.ms,
		}),
		pipeline.WithExtActions(map[string]pipeline.ActionFactory{
			"http_fetch":   ops.NewHttpFetch(),
			"prom_counter": ops.NewPromCounter(),
			"prom_gauge":   ops.NewPromGauge(),
		}),
	)
	p.lastErr.Set(0)
	for k, v := range p.gc.Targets {
		start := time.Now()
		p.l.Debug("Processing target", "name", k, "steps", v.Steps)
		po := &pipeline.PipelineOp{}
		po.ActionSpec.Children = v.Steps

		// setup initial variables
		applyVars(p.gc.Vars, gd)
		applyVars(v.Vars, gd)

		err := ex.Execute(po)
		if err != nil {
			p.l.Error("Error executing pipeline", "name", k, "err", err)
			p.lastErr.Set(1)
			p.scrapeFailures.WithLabelValues(k).Inc()
		}
		p.scrapeSum.WithLabelValues(k).Observe(time.Since(start).Seconds())
	}

	p.scrapeSum.Collect(ch)
	p.lastErr.Collect(ch)
	p.scrapeFailures.Collect(ch)
	p.up.Collect(ch)
	p.ms.Collect(ch)
	p.hcs.Collect(ch)
}

func NewExporter(cfg *types.Config, logger *slog.Logger, hcs types.HttpClientService, ms types.MetricService) prometheus.Collector {
	return &pipelineCollector{
		gc:  cfg,
		l:   logger,
		hcs: hcs,
		ms:  ms,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: types.PromNamespace,
			Name:      "up",
			Help:      "Indicates whether the exporter is operational",
		}),
		lastErr: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: types.PromNamespace,
			Name:      "last_error",
			Help:      "Indicates if last scrape resulted in error",
		}),
		scrapeFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: types.PromNamespace,
			Name:      "scrape_failure_count",
			Help:      "Number of failure for each target",
		}, []string{"target"}),
		scrapeSum: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: types.PromNamespace,
			Name:      "scrape_duration",
			Help:      "Summary of scrape operation for each target",
		}, []string{"target"}),
	}
}
