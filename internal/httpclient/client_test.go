package httpclient

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOnServerError(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	c, err := New(ts.URL, 5*time.Second, jar, "test-agent")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	resp, err := c.Get(context.Background(), "/", "")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls=%d want=3", got)
	}
}
