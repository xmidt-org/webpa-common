package client

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/davecgh/go-spew/spew"
)

type Handler struct {
	name string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w = httptest.NewRecorder()
	w.Header().Set("Test", req.Header.Get("Test"))
}

func TestRetryTransact(t *testing.T) {
	t.Run("RetryWithOptions", testWithRetryTransactorOptions)
	t.Run("NoRetryTransactorOptions", testWithNoRetryTransactorOptions)
}

func testWithRetryTransactorOptions(t *testing.T) {
	var (
		transactor = http.DefaultClient.Do
		om         = new(OutboundMeasures)
		client     = NewWebPAClient(*om, transactor)
		handler    = &Handler{name: "test"}
		server     = httptest.NewServer(handler)
		req, _     = http.NewRequest("GET", server.URL, nil)
		ro         = &xhttp.RetryOptions{Retries: 10}
	)

	if _, err := client.RetryTransact(req, ro); err != nil {
		t.Fatalf("%v", err)
	}
}

func testWithNoRetryTransactorOptions(t *testing.T) {
	var (
		transactor = http.DefaultClient.Do
		om         = new(OutboundMeasures)
		client     = NewWebPAClient(*om, transactor)
		handler    = &Handler{name: "test"}
		server     = httptest.NewServer(handler)
		req, _     = http.NewRequest("GET", server.URL, nil)
	)

	if _, err := client.RetryTransact(req, nil); err == nil {
		t.Fatalf("Error should not be nil")
	}
}

func TestTransact(t *testing.T) {
	var (
		transactor = http.DefaultClient.Do
		om         = new(OutboundMeasures)
		client     = NewWebPAClient(*om, transactor)
		handler    = &Handler{name: "test"}
		server     = httptest.NewServer(handler)
		req, _     = http.NewRequest("GET", server.URL, nil)
	)

	if _, err := client.Transact(req); err != nil {
		t.Fatalf("Client should of transacted correctly")
	}
}

func TestChangingTheTransactor(t *testing.T) {
	t.Run("TestWithTransactor", testChangeTransactor)
}

func testChangeTransactor(t *testing.T) {
	var (
		transactor = http.DefaultClient.Do
		om         = new(OutboundMeasures)
		client     = NewWebPAClient(*om, transactor)
		handler    = &Handler{name: "test"}
		server     = httptest.NewServer(handler)
		req, _     = http.NewRequest("GET", server.URL, nil)
		h          = make(map[string][]string)
	)

	h["Test"] = []string{"1"}
	req.Header = h

	decorator := func(t func(r *http.Request) (*http.Response, error)) func(*http.Request) (*http.Response, error) {
		return func(r *http.Request) (*http.Response, error) {
			r.Header.Set("Test", "2")
			return t(r)
		}
	}

	client.ChangeTransactor(decorator(transactor))
	res, _ := client.Transact(req)
	res.Header.Get("Test")

	if !reflect.DeepEqual(req.Header.Get("Test"), res.Request.Header.Get("Test")) {
		t.Fatalf("Header should equal: %v, Got: %v", spew.Sdump(req.Header.Get("Test")), spew.Sdump(res.Request.Header.Get("Test")))
	}
}
