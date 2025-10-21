package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"spoutmc/internal/models"
	"strings"

	"spoutmc/internal/log"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------

var logger = log.GetLogger()

// ------------------------------ Load Options -------------------------------- //

type LoadOptions struct {
	RepoUrl      string
	Subdir       string
	IncludeGlobs []string // default: **/*.yaml, **/*.yml
	ExcludeGlobs []string // default: **/.git/**, **/node_modules/**
	PAT          string
	Username     string
}

// ------------------------------- Public API --------------------------------- //

// LoadConfiguration clones the repo into memory, discovers YAML files, parses,
// and assembles a single SpoutConfiguration.
func LoadConfiguration(opts LoadOptions) (*models.SpoutConfiguration, error) {
	if opts.RepoUrl == "" {
		return nil, errors.New("RepoUrl is required")
	}
	if len(opts.IncludeGlobs) == 0 {
		opts.IncludeGlobs = []string{"**/*.yaml", "**/*.yml"}
	}
	if len(opts.ExcludeGlobs) == 0 {
		opts.ExcludeGlobs = []string{"**/.git/**", "**/node_modules/**"}
	}

	// Clone in-memory
	fs := memfs.New()
	storer := memory.NewStorage()
	co := &git.CloneOptions{URL: opts.RepoUrl}
	if opts.PAT != "" {
		user := opts.Username
		if user == "" {
			user = "spoutmc"
		}
		co.Auth = &http.BasicAuth{Username: user, Password: opts.PAT}
		logger.Info("Setting up auth for git clone")
	}
	if _, err := git.Clone(storer, fs, co); err != nil {
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	// Discover files
	root := "/"
	if opts.Subdir != "" {
		root = "/" + strings.TrimPrefix(filepath.ToSlash(opts.Subdir), "/")
	}
	files, err := discoverFiles(fs, root, opts.IncludeGlobs, opts.ExcludeGlobs)
	if err != nil {
		return nil, fmt.Errorf("discover files: %w", err)
	}

	// Parse & assemble configuration
	acc := &models.SpoutConfiguration{}
	for _, f := range files {
		raw, err := readAll(fs, f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		docIndex := 0
		for {
			var node any
			if err := dec.Decode(&node); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, fmt.Errorf("yaml decode %s: %w", f, err)
			}
			if node == nil {
				continue
			}
			// Re-marshal the single document so we can unmarshal into typed structs.
			buf, err := yaml.Marshal(node)
			if err != nil {
				return nil, fmt.Errorf("yaml marshal doc %s#%d: %w", f, docIndex, err)
			}
			if err := mergeDocIntoConfig(acc, buf); err != nil {
				return nil, fmt.Errorf("merge doc %s#%d: %w", f, docIndex, err)
			}
			docIndex++
		}
	}

	// Final pass: normalize & validate
	if err := finalizeSpoutConfiguration(acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// ------------------------------ Merge logic --------------------------------- //

// mergeDocIntoConfig tries several shapes:
// 1) Whole SpoutConfiguration document
// 2) A SpoutContainerNetwork document (fields "name"/"driver")
// 3) A single SpoutServer or an array of SpoutServer under "servers"
func mergeDocIntoConfig(dst *models.SpoutConfiguration, ydoc []byte) error {
	// 1) unmarshal into a generic map
	var raw any
	if err := yaml.Unmarshal(ydoc, &raw); err != nil {
		return err
	}
	if raw == nil {
		return nil
	}

	// 2) normalize keys recursively to kebab-case
	norm := normalizeKeys(raw)

	// 3) re-marshal the normalized doc
	buf, err := yaml.Marshal(norm)
	if err != nil {
		return err
	}

	// 4) run the original merge attempts on the normalized YAML
	// --- whole config
	var whole models.SpoutConfiguration
	if err := yaml.Unmarshal(buf, &whole); err == nil && (len(whole.Servers) > 0) {
		mergeWhole(dst, &whole)
		return nil
	}
	// --- servers list
	var withServers struct {
		Servers []models.SpoutServer `yaml:"servers"`
	}
	if err := yaml.Unmarshal(buf, &withServers); err == nil && len(withServers.Servers) > 0 {
		for i := range withServers.Servers {
			addOrReplaceServer(dst, withServers.Servers[i])
		}
		return nil
	}
	// --- single server
	var single models.SpoutServer
	if err := yaml.Unmarshal(buf, &single); err == nil && (single.Name != "" || single.Image != "") {
		addOrReplaceServer(dst, single)
		return nil
	}
	return nil
}

// normalizeKeys converts map keys to kebab-case (e.g. Servers -> servers, containerNetwork -> container-network)
func normalizeKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			nk := toKebab(k)
			out[nk] = normalizeKeys(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			nk := toKebab(fmt.Sprint(k))
			out[nk] = normalizeKeys(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = normalizeKeys(t[i])
		}
		return t
	default:
		return t
	}
}

var upperAcr = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
var lowerUp = regexp.MustCompile(`([a-z0-9])([A-Z])`)
var spaces = regexp.MustCompile(`[\s_]+`)

// toKebab turns "Servers", "server", "containerNetwork", "container-network", "container_network"
// into "servers", "server", "container-network"
func toKebab(s string) string {
	s = strings.TrimSpace(s)
	s = spaces.ReplaceAllString(s, "-")
	// split acronyms and lowerCamel: "HTTPServer"->"HTTP-Server", then normalize
	s = upperAcr.ReplaceAllString(s, "${1}-${2}")
	s = lowerUp.ReplaceAllString(s, "${1}-${2}")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "--", "-")
	return s
}

func mergeWhole(dst, src *models.SpoutConfiguration) {
	for _, s := range src.Servers {
		addOrReplaceServer(dst, s)
	}
}

func addOrReplaceServer(cfg *models.SpoutConfiguration, srv models.SpoutServer) {
	if srv.Env == nil {
		srv.Env = models.StringMap{}
	}
	// dedupe by Name (last one wins)
	for i := range cfg.Servers {
		if strings.EqualFold(cfg.Servers[i].Name, srv.Name) && srv.Name != "" {
			cfg.Servers[i] = mergeServer(cfg.Servers[i], srv)
			return
		}
	}
	cfg.Servers = append(cfg.Servers, srv)
}

func mergeServer(a, b models.SpoutServer) models.SpoutServer {
	if b.Name != "" {
		a.Name = b.Name
	}
	if b.Image != "" {
		a.Image = b.Image
	}
	if b.Proxy {
		a.Proxy = true
	}
	if b.Lobby {
		a.Lobby = true
	}
	if b.EnvID != 0 {
		a.EnvID = b.EnvID
	}
	if b.Env != nil {
		if a.Env == nil {
			a.Env = models.StringMap{}
		}
		for k, v := range b.Env {
			a.Env[k] = v
		}
	}
	if b.PortsID != 0 {
		a.PortsID = b.PortsID
	}
	if b.Port != 0 {
		a.Port = b.Port
	}
	if len(b.Ports) > 0 {
		a.Ports = append(a.Ports, b.Ports...)
	}
	if len(b.Volumes) > 0 {
		a.Volumes = append(a.Volumes, b.Volumes...)
	}
	return a
}

// ------------------------------ Finalization -------------------------------- //

func finalizeSpoutConfiguration(cfg *models.SpoutConfiguration) error {
	// Sort servers by name for stable output
	sort.SliceStable(cfg.Servers, func(i, j int) bool {
		return strings.ToLower(cfg.Servers[i].Name) < strings.ToLower(cfg.Servers[j].Name)
	})
	// Basic validation
	for _, s := range cfg.Servers {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("server is missing name")
		}
		if strings.TrimSpace(s.Image) == "" {
			return fmt.Errorf("server %q is missing image", s.Name)
		}
	}
	return nil
}

// ------------------------------ FS helpers ---------------------------------- //

func readAll(fs billy.Filesystem, p string) ([]byte, error) {
	f, err := fs.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func discoverFiles(fs billy.Filesystem, root string, include, exclude []string) ([]string, error) {
	var out []string
	err := walk(fs, root, func(p string, isDir bool) error {
		if isDir {
			for _, ex := range exclude {
				if matchGlob(ex, p+"/") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		for _, ex := range exclude {
			if matchGlob(ex, p) {
				return nil
			}
		}
		for _, inc := range include {
			if matchGlob(inc, p) {
				out = append(out, p)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return uniq(out), nil
}

func walk(fs billy.Filesystem, start string, fn func(p string, isDir bool) error) error {
	if start == "" {
		start = "/"
	}
	info, err := fs.Stat(start)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fn(start, false)
	}
	stack := []string{start}
	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if err := fn(dir, true); err != nil {
			if errors.Is(err, filepath.SkipDir) {
				continue
			}
			return err
		}
		ents, err := fs.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range ents {
			p := path.Join(dir, e.Name())
			if e.IsDir() {
				stack = append(stack, p)
			} else {
				if err := fn(p, false); err != nil {
					if errors.Is(err, filepath.SkipDir) {
						continue
					}
					return err
				}
			}
		}
	}
	return nil
}

func matchGlob(pattern, name string) bool {
	pp := filepath.ToSlash(pattern)
	n := filepath.ToSlash(strings.TrimPrefix(name, "/"))
	// minimal ** support
	if strings.Contains(pp, "**") {
		parts := strings.Split(pp, "**")
		if len(parts) == 2 {
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(n, strings.TrimSuffix(prefix, "/")) &&
				strings.HasSuffix(n, strings.TrimPrefix(suffix, "/"))
		}
	}
	ok, _ := filepath.Match(pp, n)
	return ok
}

func uniq(ss []string) []string {
	m := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, seen := m[s]; seen {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
