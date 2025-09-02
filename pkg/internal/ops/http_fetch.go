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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-pipeline/pkg/pipeline"
	te "github.com/rkosegi/yaml-pipeline/pkg/pipeline/template_engine"
	"github.com/rkosegi/yaml-toolkit/dom"
	"github.com/rkosegi/yaml-toolkit/props"
	"github.com/samber/lo"
)

var pp = props.NewPathParser()

func NewHttpFetch() pipeline.ActionFactory {
	return &httpFetchOpFactory{}
}

type (
	httpFetchOpFactory struct{}

	httpFetchOp struct {
		// Method is HTTP method to use, when omitted, then GET is assumed
		Method *string

		// Url is location to fetch. Template is supported.
		Url string

		// Headers are optional map of headers to send.
		// Names and values can use template.
		Headers map[string]string

		// ParseJson flag indicating whether to parse response body as JSON into data tree
		ParseJson *bool `yaml:"parseJson,omitempty"`

		// StoreTo is path within the global data where parsed response is stored
		StoreTo string `yaml:"storeTo"`
	}
)

func (h *httpFetchOpFactory) ForArgs(ctx pipeline.ClientContext, args pipeline.StrKeysAnyValues) pipeline.Action {
	r := &httpFetchOp{}
	pipeline.ApplyArgs(ctx, r, args)
	return r
}

func (h *httpFetchOp) String() string {
	return fmt.Sprintf("HttpFetch[Method: %v, Url: %s]", h.Method, h.Url)
}

func (h *httpFetchOp) doWithResponse(ctx pipeline.ActionContext, resp *types.ParsedHttpResponse) error {
	c := dom.ContainerNode()
	c.AddValue("status", dom.LeafNode(resp.StatusCode))
	c.AddValue("body", dom.LeafNode(string(resp.Body)))
	hc := c.AddContainer("headers")
	for k, v := range h.Headers {
		hc.AddValue(k, dom.LeafNode(v))
	}
	if h.ParseJson != nil && *h.ParseJson {
		if jd, err := dom.DecodeReader(bytes.NewReader(resp.Body), dom.DefaultJsonDecoder); err != nil {
			return err
		} else {
			c.AddValue("json", jd)
		}
	}
	ctx.Data().Set(pp.MustParse(h.StoreTo), c)
	ctx.InvalidateSnapshot()
	return nil
}

func (h *httpFetchOp) Do(ctx pipeline.ActionContext) error {
	var (
		err        error
		data       []byte
		req        *http.Request
		resp       *http.Response
		cachedResp *types.ParsedHttpResponse
		hcs        types.HttpClientService
	)
	ss := ctx.Snapshot()
	m := http.MethodGet
	if h.Method != nil {
		m = *h.Method
	}
	url := ctx.TemplateEngine().RenderLenient(h.Url, ss)
	if svc := ctx.Ext().GetService("HttpClient"); svc == nil {
		return errors.New("service not found: HttpClient")
	} else {
		hcs = svc.(types.HttpClientService)
	}

	if req, err = http.NewRequest(m, url, nil); err != nil {
		return err
	}

	for k, v := range h.Headers {
		req.Header.Set(k, ctx.TemplateEngine().RenderLenient(v, ss))
	}

	if resp, err = hcs.RoundTripper().RoundTrip(req); err != nil {
		return err
	}

	if data, err = io.ReadAll(resp.Body); err != nil {
		return err
	}

	cachedResp = &types.ParsedHttpResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       data,
	}
	ctx.Logger().Log("status", resp.Status)
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
