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

package pipeline_ext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rkosegi/universal-exporter/pkg/internal/cache"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/dom"
	"github.com/rkosegi/yaml-toolkit/pipeline"
	te "github.com/rkosegi/yaml-toolkit/pipeline/template_engine"
	"github.com/samber/lo"
)

const defHttpTimeout = 15 * time.Second

func NewFactory() pipeline.ActionFactory {
	return &httpFetchOpFactory{}
}

type (
	httpFetchOpFactory struct {
		ac pipeline.ActionContext
	}

	httpFetchOp struct {
		// Method is HTTP method to use, when omitted, then GET is assumed
		Method *string

		// Url is location to fetch. Template is supported.
		Url string

		// Headers are optional map of headers to send.
		// Names and values can use template.
		Headers map[string]string

		// Timeout is overall timeout after which request will fail
		Timeout *time.Duration

		// ParseJson flag indicating whether to parse response body as JSON into data tree
		ParseJson *bool `yaml:"parseJson,omitempty"`

		// StoreTo is path within the global data where parsed response is stored
		StoreTo string `yaml:"storeTo"`

		// SkipCache whether to disable caching of HTTP responses.
		// When caching is not disabled, then HttpResponseCache service must be registered with
		// pipeline.
		SkipCache bool
	}
)

func (h *httpFetchOpFactory) NewForArgs(args pipeline.StrKeysAnyValues) pipeline.Action {
	r := &httpFetchOp{}
	pipeline.ApplyArgs(h.ac, r, args)
	return r
}

func (h *httpFetchOp) String() string {
	return fmt.Sprintf("HttpFetch[Method: %v, Url: %s]", h.Method, h.Url)
}

func (h *httpFetchOp) doWithResponse(ctx pipeline.ActionContext, resp *types.ParsedHttpResponse) error {
	c := ctx.Factory().Container()
	c.AddValue("status", dom.LeafNode(resp.StatusCode))
	c.AddValue("body", dom.LeafNode(string(resp.Body)))
	hc := c.AddContainer("headers")
	for k, v := range h.Headers {
		hc.AddValue(k, dom.LeafNode(v))
	}
	if h.ParseJson != nil && *h.ParseJson {
		if jd, err := ctx.Factory().FromReader(bytes.NewReader(resp.Body), dom.DefaultJsonDecoder); err != nil {
			return err
		} else {
			c.AddValue("json", jd)
		}
	}
	ctx.Data().AddValueAt(h.StoreTo, c)
	return nil
}

func (h *httpFetchOp) Do(ctx pipeline.ActionContext) error {
	var (
		err        error
		body       bytes.Buffer
		resp       *http.Response
		cachedResp *types.ParsedHttpResponse
		hrc        types.HttpResponseCache
	)
	hrc = cache.NoopCache
	ss := ctx.Snapshot()
	timeout := defHttpTimeout
	m := http.MethodGet
	if h.Method != nil {
		m = *h.Method
	}
	if h.Timeout != nil {
		timeout = *h.Timeout
	}
	url := ctx.TemplateEngine().RenderLenient(h.Url, ss)

	if !h.SkipCache {
		if svc := ctx.Ext().GetService("HttpResponseCache"); svc != nil {
			return errors.New("service not found: HttpResponseCache")
		} else {
			hrc = svc.(types.HttpResponseCache)
		}
	}

	if cachedResp = hrc.Get(url); cachedResp != nil {
		ctx.Logger().Log("using cached response", url)
		return h.doWithResponse(ctx, cachedResp)
	}

	c, cancel := context.WithCancel(context.TODO())
	timer := time.AfterFunc(timeout, func() {
		cancel()
	})
	defer timer.Stop()

	req, err := http.NewRequest(m, url, nil)
	if err != nil {
		return err
	}
	for k, v := range h.Headers {
		req.Header.Set(k, ctx.TemplateEngine().RenderLenient(v, ss))
	}
	req = req.WithContext(c)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			ctx.Logger().Log("unable to close response body", err.Error())
		}
	}(resp.Body)
	_, err = io.Copy(&body, resp.Body)
	if err != nil {
		return err
	}
	cachedResp = &types.ParsedHttpResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body.Bytes(),
	}

	hrc.Put(req.URL.String(), cachedResp)

	return h.doWithResponse(ctx, cachedResp)
}

func (h *httpFetchOp) CloneWith(ctx pipeline.ActionContext) pipeline.Action {
	ss := ctx.Snapshot()
	return &httpFetchOp{
		Method:  lo.ToPtr(ctx.TemplateEngine().RenderLenient(strOrDef(h.Method, http.MethodGet), ss)),
		Url:     ctx.TemplateEngine().RenderLenient(h.Url, ss),
		Headers: renderMapStrStr(h.Headers, ctx.TemplateEngine(), ss),
	}
}

type StrKeyValues map[string]string

// AsUntypedMap converts this map to map[string]interface{} suitable for template related operations.
func (skv StrKeyValues) AsUntypedMap() map[string]interface{} {
	out := make(map[string]interface{}, len(skv))
	for k, v := range skv {
		out[k] = v
	}
	return out
}

// RenderValues renders values of this map using provided TemplateEngine and data.
func (skv StrKeyValues) RenderValues(teng te.TemplateEngine, data map[string]interface{}) map[string]string {
	out := make(map[string]string, len(skv))
	for k, v := range skv {
		out[k] = teng.RenderLenient(v, data)
	}
	return out
}

func renderMapStrStr(in map[string]string, teng te.TemplateEngine, data map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = teng.RenderLenient(v, data)
	}
	return out
}

func strOrDef(pstr *string, def string) string {
	if pstr == nil {
		return def
	}
	return *pstr
}
