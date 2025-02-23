package cmdio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name           string
	v              any
	outputFormat   flags.Output
	headerTemplate string
	template       string
	expected       string
	errMessage     string
}

var dummyWorkspace1 = provisioning.Workspace{
	WorkspaceId:   123,
	WorkspaceName: "abc",
}

var dummyWorkspace2 = provisioning.Workspace{
	WorkspaceId:   456,
	WorkspaceName: "def",
}

type dummyIterator struct {
	items []*provisioning.Workspace
}

func (d *dummyIterator) HasNext(_ context.Context) bool {
	return len(d.items) > 0
}

func (d *dummyIterator) Next(ctx context.Context) (*provisioning.Workspace, error) {
	if !d.HasNext(ctx) {
		return nil, errors.New("no more items")
	}
	item := d.items[0]
	d.items = d.items[1:]
	return item, nil
}

func makeWorkspaces(count int) []*provisioning.Workspace {
	res := make([]*provisioning.Workspace, 0, count)
	next := []*provisioning.Workspace{&dummyWorkspace1, &dummyWorkspace2}
	for range count {
		n := next[0]
		next = append(next[1:], n)
		res = append(res, n)
	}
	return res
}

func makeIterator(count int) listing.Iterator[*provisioning.Workspace] {
	items := make([]*provisioning.Workspace, 0, count)
	items = append(items, makeWorkspaces(count)...)
	return &dummyIterator{
		items: items,
	}
}

func makeBigOutput(count int) string {
	res := bytes.Buffer{}
	for _, ws := range makeWorkspaces(count) {
		res.WriteString(fmt.Sprintf("%d  %s\n", ws.WorkspaceId, ws.WorkspaceName))
	}
	return res.String()
}

func must[T any](a T, e error) T {
	if e != nil {
		panic(e)
	}
	return a
}

var testCases = []testCase{
	{
		name:           "Workspace with header and template",
		v:              dummyWorkspace1,
		outputFormat:   flags.OutputText,
		headerTemplate: "id\tname",
		template:       "{{.WorkspaceId}}\t{{.WorkspaceName}}",
		expected: `id   name
123  abc`,
	},
	{
		name:         "Workspace with no header and template",
		v:            dummyWorkspace1,
		outputFormat: flags.OutputText,
		template:     "{{.WorkspaceId}}\t{{.WorkspaceName}}",
		expected:     `123  abc`,
	},
	{
		name:         "Workspace with no header and no template",
		v:            dummyWorkspace1,
		outputFormat: flags.OutputText,
		expected: `{
  "workspace_id":123,
  "workspace_name":"abc"
}
`,
	},
	{
		name:           "Workspace Iterator with header and template",
		v:              makeIterator(2),
		outputFormat:   flags.OutputText,
		headerTemplate: "id\tname",
		template:       "{{range .}}{{.WorkspaceId}}\t{{.WorkspaceName}}\n{{end}}",
		expected: `id   name
123  abc
456  def
`,
	},
	{
		name:         "Workspace Iterator with no header and template",
		v:            makeIterator(2),
		outputFormat: flags.OutputText,
		template:     "{{range .}}{{.WorkspaceId}}\t{{.WorkspaceName}}\n{{end}}",
		expected: `123  abc
456  def
`,
	},
	{
		name:         "Workspace Iterator with no header and no template",
		v:            makeIterator(2),
		outputFormat: flags.OutputText,
		expected:     string(must(json.MarshalIndent(makeWorkspaces(2), "", "  "))) + "\n",
	},
	{
		name:           "Big Workspace Iterator with template",
		v:              makeIterator(234),
		outputFormat:   flags.OutputText,
		headerTemplate: "id\tname",
		template:       "{{range .}}{{.WorkspaceId}}\t{{.WorkspaceName}}\n{{end}}",
		expected:       "id   name\n" + makeBigOutput(234),
	},
	{
		name:         "Big Workspace Iterator with no template",
		v:            makeIterator(234),
		outputFormat: flags.OutputText,
		expected:     string(must(json.MarshalIndent(makeWorkspaces(234), "", "  "))) + "\n",
	},
	{
		name:         "io.Reader",
		v:            strings.NewReader("a test"),
		outputFormat: flags.OutputText,
		expected:     "a test",
	},
	{
		name:         "io.Reader",
		v:            strings.NewReader("a test"),
		outputFormat: flags.OutputJSON,
		errMessage:   "json output not supported",
	},
}

func TestRender(t *testing.T) {
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			ctx := context.Background()
			cmdIO := NewIO(ctx, c.outputFormat, nil, output, output, c.headerTemplate, c.template)
			ctx = InContext(ctx, cmdIO)
			var err error
			if vv, ok := c.v.(listing.Iterator[*provisioning.Workspace]); ok {
				err = RenderIterator(ctx, vv)
			} else {
				err = Render(ctx, c.v)
			}
			if c.errMessage != "" {
				assert.ErrorContains(t, err, c.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expected, output.String())
			}
		})
	}
}
