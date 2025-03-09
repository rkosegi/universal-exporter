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

package internal

import (
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/rkosegi/yaml-toolkit/dom"
	"github.com/rkosegi/yaml-toolkit/pipeline"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func dumpDom(t *testing.T, doc dom.ContainerBuilder) {
	doc.Walk(func(path string, parent dom.ContainerBuilder, node dom.Node) bool {
		t.Logf("%s -> %T", path, node)
		return true
	})

}

func TestXml2Dom(t *testing.T) {
	var (
		node *html.Node
		// nodes []*html.Node
		err error
	)
	node, err = htmlquery.LoadDoc("../../testdata/shmu1.html")
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// nodes, err = htmlquery.QueryAll(node, "//table[@id='tab-aktualne-pocasie']/tbody/tr")
	// assert.NoError(t, err)
	// assert.NotNil(t, nodes)
	// assert.Equal(t, 99, len(nodes))
	// assert.Equal(t, "tr", nodes[0].Data)
	b := dom.Builder().Container()

	pp := pipeline.New(pipeline.WithData(nil))

	op := &pipeline.Html2DomOp{
		From: "foo",
	}
	pp.Execute(op)

	dumpDom(t, b)
	b.Children()
}
