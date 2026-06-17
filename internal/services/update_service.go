package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/embrionix/dashboard/internal/config"
	"github.com/embrionix/dashboard/internal/version"
	"github.com/embrionix/dashboard/pkg/logger"
	"github.com/minio/selfupdate"
	"go.uber.org/zap"
)

// UpdateService checks GitHub Releases for a newer version and can self-update
// the running binary (download the matching asset, swap it in place, restart).
type UpdateService struct {
	cfg      config.UpdatesConfig
	http     *http.Client
	execPath string // captured at startup; points at the original binary path

	mu     sync.RWMutex
	latest UpdateStatus
}

// UpdateStatus is the cached result of the most recent release check.
type UpdateStatus struct {
	CurrentVersion  string    `json:"current_version"`
	LatestVersion   string    `json:"latest_version"`
	UpdateAvailable bool      `json:"update_available"`
	ReleaseURL      string    `json:"release_url"`
	ReleaseNotes    string    `json:"release_notes"`
	CheckedAt       time.Time `json:"checked_at"`
	Enabled         bool      `json:"enabled"`
	Error           string    `json:"error,omitempty"`
}

// NewUpdateService captures the current executable path (used later to relaunch).
func NewUpdateService(cfg config.UpdatesConfig) *UpdateService {
	exe, err := os.Executable()
	if err != nil {
		logger.Warn("update: could not determine executable path; self-update disabled", zap.Error(err))
	}
	return &UpdateService{
		cfg:      cfg,
		http:     &http.Client{Timeout: 30 * time.Second},
		execPath: exe,
		latest: UpdateStatus{
			CurrentVersion: version.Version,
			Enabled:        cfg.Enabled,
		},
	}
}

// Status returns the last cached release-check result.
func (s *UpdateService) Status() UpdateStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
}

// StartChecker runs an initial check and then re-checks on the configured
// interval until ctx is cancelled. No-op when updates are disabled.
func (s *UpdateService) StartChecker(ctx context.Context) {
	if !s.cfg.Enabled || s.cfg.Repo == "" {
		return
	}
	interval := time.Duration(s.cfg.CheckIntervalHours) * time.Hour
	if interval < time.Hour {
		interval = time.Hour
	}
	go func() {
		s.Check(ctx)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.Check(ctx)
			}
		}
	}()
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Check queries GitHub for the latest release and updates the cached status.
func (s *UpdateService) Check(ctx context.Context) UpdateStatus {
	status := UpdateStatus{
		CurrentVersion: version.Version,
		Enabled:        s.cfg.Enabled,
		CheckedAt:      time.Now(),
	}

	rel, err := s.latestRelease(ctx)
	if err != nil {
		status.Error = err.Error()
		logger.Warn("update: release check failed", zap.Error(err))
		s.store(status)
		return status
	}

	status.LatestVersion = rel.TagName
	status.ReleaseURL = rel.HTMLURL
	status.ReleaseNotes = rel.Body
	// Only a real (tagged) build can be meaningfully compared to a release tag.
	status.UpdateAvailable = version.IsRelease() && isNewer(rel.TagName, version.Version)

	s.store(status)
	return status
}

func (s *UpdateService) store(st UpdateStatus) {
	s.mu.Lock()
	s.latest = st
	s.mu.Unlock()
}

func (s *UpdateService) latestRelease(ctx context.Context) (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", s.cfg.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases returned HTTP %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// assetName is the release asset filename for this platform, e.g.
// "embrionix-dashboard-windows-amd64.exe".
func assetName() string {
	name := fmt.Sprintf("embrionix-dashboard-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// Apply downloads the matching release binary, verifies its checksum (when a
// checksums.txt asset is present), swaps the running binary, and relaunches.
// It blocks through the swap; the relaunch + exit happen in a background
// goroutine so the HTTP response can flush first.
func (s *UpdateService) Apply(ctx context.Context) error {
	if !s.cfg.Enabled {
		return fmt.Errorf("updates are disabled")
	}
	if s.execPath == "" {
		return fmt.Errorf("self-update unavailable: executable path unknown")
	}

	rel, err := s.latestRelease(ctx)
	if err != nil {
		return err
	}
	if !isNewer(rel.TagName, version.Version) {
		return fmt.Errorf("already up to date (%s)", version.Version)
	}

	want := assetName()
	var binURL, sumURL string
	for _, a := range rel.Assets {
		switch {
		case a.Name == want:
			binURL = a.BrowserDownloadURL
		case a.Name == "checksums.txt":
			sumURL = a.BrowserDownloadURL
		}
	}
	if binURL == "" {
		return fmt.Errorf("no release asset %q for this platform", want)
	}

	bin, err := s.download(ctx, binURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Verify checksum when available; refuse the update on mismatch.
	if sumURL != "" {
		sums, err := s.download(ctx, sumURL)
		if err == nil {
			if expected := checksumFor(string(sums), want); expected != "" {
				got := sha256.Sum256(bin)
				if !strings.EqualFold(expected, hex.EncodeToString(got[:])) {
					return fmt.Errorf("checksum mismatch for %s — update aborted", want)
				}
				logger.Info("update: checksum verified", zap.String("asset", want))
			}
		}
	}

	if err := selfupdate.Apply(strings.NewReader(string(bin)), selfupdate.Options{}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			logger.Error("update: rollback also failed", zap.Error(rerr))
		}
		return fmt.Errorf("apply: %w", err)
	}

	logger.Info("update: binary swapped, relaunching", zap.String("version", rel.TagName))
	go s.relaunch()
	return nil
}

// relaunch restarts the process onto the freshly-swapped binary. In "exit" mode
// it simply exits and relies on a service manager (systemd/NSSM) to restart it;
// otherwise it spawns a fresh instance itself (for unsupervised runs). The new
// instance binds the port with a retry, tolerating the brief overlap.
func (s *UpdateService) relaunch() {
	time.Sleep(1500 * time.Millisecond) // let the HTTP response flush

	if strings.EqualFold(s.cfg.RestartMode, "exit") {
		logger.Info("update: exiting for service-manager restart (restart_mode=exit)")
		os.Exit(0)
	}

	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}
	cmd := exec.Command(s.execPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		logger.Error("update: failed to relaunch — manual restart required", zap.Error(err))
		return
	}
	logger.Info("update: new instance started; exiting old process")
	os.Exit(0)
}

func (s *UpdateService) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Larger timeout for the binary download.
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// checksumFor finds the hex digest for filename within a `sha256  filename`
// style checksums listing.
func checksumFor(contents, filename string) string {
	for _, line := range strings.Split(contents, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && (f[1] == filename || strings.TrimPrefix(f[1], "*") == filename) {
			return f[0]
		}
	}
	return ""
}

// isNewer reports whether semantic version a is greater than b (both like
// "v1.2.3"; pre-release/build metadata is ignored).
func isNewer(a, b string) bool {
	pa, ok1 := parseSemver(a)
	pb, ok2 := parseSemver(b)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseSemver(s string) ([3]int, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "v"))
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
