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

package ops

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-pipeline/pkg/pipeline"
)

type (
	commonMetricOpSpec struct {
		Ref    string   `yaml:"ref"`
		Labels []string `yaml:"labels"`
	}
	promCounterVecOp struct {
		commonMetricOpSpec `yaml:",inline"`
		IncBy              *pipeline.ValOrRef `yaml:"incBy,omitempty"`
	}
	promGaugeVecOp struct {
		commonMetricOpSpec `yaml:",inline"`
		Value              pipeline.ValOrRef `yaml:"value"`
	}
)

// gauge

func (p *promGaugeVecOp) String() string {
	return fmt.Sprintf("PromGaugeOp[ref=%s,value=%v]", p.Ref, p.Value)
}

func (p *promGaugeVecOp) Do(ctx pipeline.ActionContext) error {
	if len(p.Ref) == 0 {
		return errors.New("empty metric reference")
	}

	var (
		svc interface{}
		val float64
		ms  types.MetricService
	)

	if svc = ctx.Ext().GetService("MetricService"); svc == nil {
		return errors.New("no such service: MetricService")
	}

	ms = svc.(types.MetricService)

	spec, err := ms.GetRef(p.Ref)
	if err != nil {
		return err
	}

	val, err = strconv.ParseFloat(p.Value.Resolve(ctx), 64)
	if err != nil {
		return err
	}

	labels := ctx.TemplateEngine().RenderSliceLenient(p.Labels, ctx.Snapshot())
	vec := spec.MetricRef.(*prometheus.GaugeVec)
	vec.WithLabelValues(labels...).Set(val)
	return nil
}

func (p *promGaugeVecOp) CloneWith(ctx pipeline.ActionContext) pipeline.Action {
	return &promGaugeVecOp{
		commonMetricOpSpec: commonMetricOpSpec{
			Ref:    ctx.TemplateEngine().RenderLenient(p.Ref, ctx.Snapshot()),
			Labels: ctx.TemplateEngine().RenderSliceLenient(p.Labels, ctx.Snapshot()),
		},
		Value: p.Value,
	}
}

// counter

func (p *promCounterVecOp) String() string {
	return fmt.Sprintf("PromCounterOp[ref=%s]", p.Ref)
}

func (p *promCounterVecOp) Do(ctx pipeline.ActionContext) error {
	var (
		svc interface{}
		ms  types.MetricService
		err error
	)
	incBy := float64(1)
	if len(p.Ref) == 0 {
		return errors.New("empty metric reference")
	}

	if p.IncBy != nil {
		incBy, err = strconv.ParseFloat(p.IncBy.Resolve(ctx), 64)
		if err != nil {
			return err
		}
	}

	if svc = ctx.Ext().GetService("MetricService"); svc != nil {
		return errors.New("no such service: MetricService")
	}

	ms = svc.(types.MetricService)

	spec, err := ms.GetRef(p.Ref)

	if err != nil {
		return err
	}

	vec := spec.MetricRef.(*prometheus.CounterVec)
	vec.WithLabelValues(p.Labels...).Add(incBy)
	return nil
}

func (p *promCounterVecOp) CloneWith(ctx pipeline.ActionContext) pipeline.Action {
	return &promCounterVecOp{
		commonMetricOpSpec: commonMetricOpSpec{
			Ref:    ctx.TemplateEngine().RenderLenient(p.Ref, ctx.Snapshot()),
			Labels: ctx.TemplateEngine().RenderSliceLenient(p.Labels, ctx.Snapshot()),
		},
		IncBy: p.IncBy,
	}
}

func NewPromCounter() pipeline.ActionFactory {
	return SimpleActionFactory[promCounterVecOp](func() *promCounterVecOp {
		return &promCounterVecOp{}
	})
}

func NewPromGauge() pipeline.ActionFactory {
	return SimpleActionFactory[promGaugeVecOp](func() *promGaugeVecOp {
		return &promGaugeVecOp{}
	})
}

type simpleActionFactoryImpl[T any] struct {
	fn func() *T
}

func (s *simpleActionFactoryImpl[T]) ForArgs(ctx pipeline.ClientContext, args pipeline.StrKeysAnyValues) pipeline.Action {
	t := s.fn()
	pipeline.ApplyArgs(ctx, t, args)
	var x interface{} = t
	return x.(pipeline.Action)
}

func SimpleActionFactory[T any](fn func() *T) pipeline.ActionFactory {
	return &simpleActionFactoryImpl[T]{fn: fn}
}
