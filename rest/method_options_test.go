package rest

import (
	"net/http"
	"testing"

	"golang.org/x/net/context"

	"github.com/cool-rest/rest-layer/resource"
	"github.com/cool-rest/rest-layer/schema"
	"github.com/stretchr/testify/assert"
)

func TestHandlerOptionsList(t *testing.T) {
	index := resource.NewIndex()
	test := index.Bind("test", schema.Schema{Fields: schema.Fields{"id": {}}}, nil, resource.DefaultConf)
	r, _ := http.NewRequest("OPTIONS", "/test", nil)
	rm := &RouteMatch{
		ResourcePath: []*ResourcePathComponent{
			&ResourcePathComponent{
				Name:     "test",
				Resource: test,
			},
		},
	}
	status, headers, body := listOptions(context.TODO(), r, rm)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, http.Header{"Allow": []string{"DELETE, GET, HEAD, POST"}}, headers)
	assert.Nil(t, body)
}
