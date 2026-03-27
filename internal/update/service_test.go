package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsNewerVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{name: "patch upgrade", current: "0.0.6", latest: "0.0.7", expected: true},
		{name: "same version", current: "0.0.7", latest: "0.0.7", expected: false},
		{name: "downgrade", current: "0.0.8", latest: "0.0.7", expected: false},
		{name: "stable greater than prerelease", current: "0.0.7-beta.1", latest: "0.0.7", expected: true},
		{name: "prerelease ordering", current: "0.0.7-beta.1", latest: "0.0.7-beta.2", expected: true},
		{name: "v prefix is accepted", current: "v0.0.6", latest: "v0.0.7", expected: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isNewerVersion(tc.current, tc.latest)
			if got != tc.expected {
				t.Fatalf("isNewerVersion(%q, %q)=%v want %v", tc.current, tc.latest, got, tc.expected)
			}
		})
	}
}

func TestAssetNameForRuntime(t *testing.T) {
	t.Parallel()

	if got := assetNameForRuntime("darwin", "arm64"); got != "spoutmc-darwin-arm64" {
		t.Fatalf("unexpected darwin asset name: %s", got)
	}
	if got := assetNameForRuntime("linux", "amd64"); got != "spoutmc-linux-amd64" {
		t.Fatalf("unexpected linux asset name: %s", got)
	}
	if got := assetNameForRuntime("windows", "amd64"); got != "spoutmc-windows-amd64.exe" {
		t.Fatalf("unexpected windows asset name: %s", got)
	}
}

func TestVerifyChecksum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	binPath := filepath.Join(dir, "spoutmc-darwin-arm64")
	sumPath := filepath.Join(dir, "spoutmc-darwin-arm64.sha256")

	content := []byte("spoutmc-test")
	if err := os.WriteFile(binPath, content, 0o644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	if err := os.WriteFile(sumPath, []byte("958a57e5650d408176a68fd09f69e52702c26823ceb1f42c58af34b1f7227ecf  spoutmc-darwin-arm64\n"), 0o644); err != nil {
		t.Fatalf("write checksum: %v", err)
	}

	if err := verifyChecksum(binPath, sumPath, "spoutmc-darwin-arm64"); err != nil {
		t.Fatalf("verifyChecksum returned error: %v", err)
	}
}

type fakeTicker struct {
	ch chan time.Time
}

func (f *fakeTicker) Chan() <-chan time.Time {
	return f.ch
}

func (f *fakeTicker) Stop() {}

func TestRunSchedulerPerformsInitialAndRecurringChecks(t *testing.T) {
	var requestCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name":"v0.0.7",
			"html_url":"https://example.com/release",
			"body":"notes",
			"assets":[]
		}`))
	}))
	defer srv.Close()

	m := &Manager{
		repo:             "owner/repo",
		githubAPIBaseURL: srv.URL,
		currentVersion:   "0.0.6",
		httpClient:       newHTTPClient(),
		checkInterval:    24 * time.Hour,
		initialDelay:     0,
		status: Status{
			Configured:       true,
			CurrentVersion:   "0.0.6",
			State:            StateIdle,
			CurrentAssetName: "spoutmc-darwin-arm64",
		},
	}

	tick := &fakeTicker{ch: make(chan time.Time, 1)}
	m.tickerFactory = func(_ time.Duration) ticker { return tick }
	m.timerFactory = time.NewTimer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		m.runScheduler(ctx)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&requestCount) < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&requestCount) < 1 {
		t.Fatal("expected startup check to run")
	}

	tick.ch <- time.Now()
	deadline = time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&requestCount) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&requestCount) < 2 {
		t.Fatal("expected recurring check to run")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop on context cancel")
	}
}
