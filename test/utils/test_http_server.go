package testutils

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/phayes/freeport"
)

type TestHttpServer struct {
	*http.ServeMux
}

func NewTestHttpServer() *TestHttpServer {
	mux := http.NewServeMux()
	return &TestHttpServer{mux}
}

// Returns the port the server is listening on.
func (s *TestHttpServer) Start(t *testing.T) int {
	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("cannot start test server: %v", err)
	}

	srvAddr := fmt.Sprintf(":%d", port)
	srv := http.Server{
		Addr:    srvAddr,
		Handler: s,
	}

	t.Cleanup(func() {
		srv.Close()
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			t.Errorf("cannot start test server: %v", err)
		}
	}()

	waitForServer(t, srvAddr)
	return port
}

func waitForServer(t *testing.T, url string) {
	backoff := 50 * time.Millisecond

	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", url, 1*time.Second)
		if err != nil {
			time.Sleep(backoff)
			continue
		}
		err = conn.Close()
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	t.Fatalf("server on URL %s not up after 10 attempts", url)
}
