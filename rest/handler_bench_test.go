package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cool-rest/rest-layer-mem"
	"github.com/cool-rest/rest-layer/resource"
	"github.com/cool-rest/rest-layer/schema"
)

func BenchmarkServeHTTP(b *testing.B) {
	i := resource.NewIndex()
	i.Bind("foo", schema.Schema{}, mem.NewHandler(), resource.DefaultConf)
	h, _ := NewHandler(i)
	r, _ := http.NewRequest("GET", "/foo", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}
