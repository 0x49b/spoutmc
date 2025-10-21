package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"spoutmc/internal/log"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	httpgit "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var logger = log.GetLogger()

// ------------------------------- Data Models ------------------------------- //

type LoadedDoc struct {
	SourcePath  string         `json:"source_path"`
	IndexInFile int            `json:"index_in_file"`
	Content     map[string]any `json:"content"`
}

func (d LoadedDoc) Kind() string {
	if v, ok := d.Content["kind"].(string); ok {
		return v
	}
	return ""
}

func (d LoadedDoc) APIVersion() string {
	if v, ok := d.Content["apiVersion"].(string); ok {
		return v
	}
	return ""
}

type InMemoryConfig struct {
	ByFile  map[string][]LoadedDoc `json:"by_file"`
	ByKind  map[string][]LoadedDoc `json:"by_kind"`
	AllDocs []LoadedDoc            `json:"all_docs"`
}

func (m *InMemoryConfig) index() {
	m.ByFile = make(map[string][]LoadedDoc)
	m.ByKind = make(map[string][]LoadedDoc)
	for _, d := range m.AllDocs {
		m.ByFile[d.SourcePath] = append(m.ByFile[d.SourcePath], d)
		k := d.Kind()
		if k != "" {
			m.ByKind[k] = append(m.ByKind[k], d)
		}
	}
}

// ------------------------------ Load Options ------------------------------- //

type LoadOptions struct {
	RepoURL      string
	Ref          string   // branch, tag, or commit (optional)
	Subdir       string   // e.g. "configs/" (optional)
	IncludeGlobs []string // default **/*.yaml, **/*.yml
	IgnoreGlobs  []string // default **/.git/**
	Validate     bool
	// Auth (for private HTTPS repos with a Personal Access Token)
	// If PAT is set, Username defaults to "git" if empty (GitHub accepts any non-empty username).
	PAT      string
	Username string
}

// ------------------------------- Main Loader -------------------------------- //

func LoadConfigRepo(opts LoadOptions) (*InMemoryConfig, error) {
	if opts.RepoURL == "" {
		return nil, errors.New("RepoURL is required")
	}
	if len(opts.IncludeGlobs) == 0 {
		opts.IncludeGlobs = []string{"**/*.yaml", "**/*.yml"}
	}
	if len(opts.IgnoreGlobs) == 0 {
		opts.IgnoreGlobs = []string{"**/.git/**", "**/node_modules/**"}
	}

	// Clone repo entirely in memory
	storer := memory.NewStorage()
	fs := memfs.New()
	co := &git.CloneOptions{
		URL: opts.RepoURL,
	}

	// Set HTTP auth if PAT provided (for private repos over HTTPS)
	if opts.PAT != "" {
		user := opts.Username
		if user == "" {
			user = "git"
		}
		co.Auth = &httpgit.BasicAuth{Username: user, Password: opts.PAT}
		logger.Info("Setting up auth for git clone")
	}
	repo, err := git.Clone(storer, fs, co)
	if err != nil {
		logger.Error("clone failes", zap.Error(err))
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	// Optionally checkout ref
	if opts.Ref != "" {
		w, err := repo.Worktree()
		if err != nil {
			logger.Error("get worktree fail", zap.Error(err))
			return nil, err
		}
		// Try resolving as branch/tag/commit
		// First fetch the ref (shallow)
		if err := repo.Fetch(&git.FetchOptions{
			Depth:    1,
			RefSpecs: []config.RefSpec{config.RefSpec("+refs/*:refs/*")},
			Tags:     git.AllTags,
			Force:    true,
			Auth: func() transport.AuthMethod {
				if opts.PAT == "" {
					return nil
				}
				user := opts.Username
				if user == "" {
					user = "git"
				}
				return &httpgit.BasicAuth{Username: user, Password: opts.PAT}
			}(),
		}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			logger.Error("fetch fail", zap.Error(err))
			return nil, fmt.Errorf("fetch ref failed: %w", err)
		}
		// Try different ref formats
		hash, err := resolveRef(repo, opts.Ref)
		if err != nil {
			logger.Error("resolve ref fail", zap.Error(err))
			return nil, fmt.Errorf("resolve ref '%s': %w", opts.Ref, err)
		}
		if err := w.Checkout(&git.CheckoutOptions{Hash: hash}); err != nil {
			logger.Error("checkout fail", zap.Error(err))
			return nil, fmt.Errorf("checkout %s: %w", opts.Ref, err)
		}
	}

	// Discover files
	root := "/"
	if opts.Subdir != "" {
		root = "/" + strings.TrimPrefix(filepath.ToSlash(opts.Subdir), "/")
	}
	files, err := discoverFiles(fs, root, opts.IncludeGlobs, opts.IgnoreGlobs)
	if err != nil {
		logger.Error("discover fail", zap.Error(err))
		return nil, err
	}

	// Parse YAML
	mem := &InMemoryConfig{}
	for _, f := range files {
		b, err := readAll(fs, f)
		if err != nil {
			logger.Error("read fail", zap.Error(err))
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		dec := yaml.NewDecoder(bytes.NewReader(b))
		idx := 0
		for {
			var anydoc any
			if err := dec.Decode(&anydoc); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, fmt.Errorf("yaml decode %s: %w", f, err)
			}
			if anydoc == nil {
				continue
			}
			m, ok := normalizeToMap(anydoc)
			if !ok {
				m = map[string]any{"_root": anydoc}
			}
			mem.AllDocs = append(mem.AllDocs, LoadedDoc{SourcePath: f, IndexInFile: idx, Content: m})
			idx++
		}
	}
	mem.index()

	if opts.Validate {
		warnings := basicValidate(mem)
		if len(warnings) > 0 {
			logger.Error("validate fail", zap.Any("warnings", warnings))
			//fmt.Fprintln(io.Discard) // placeholder: keep function pure; log elsewhere if needed
		}
	}
	return mem, nil
}

// ------------------------------ Helper funcs -------------------------------- //

func resolveRef(repo *git.Repository, ref string) (plumbing.Hash, error) {
	// Try as full ref
	if r, err := repo.Reference(plumbing.ReferenceName(ref), true); err == nil {
		return r.Hash(), nil
	}
	// Try heads/
	if r, err := repo.Reference(plumbing.NewBranchReferenceName(ref), true); err == nil {
		return r.Hash(), nil
	}
	// Try tags/
	if r, err := repo.Reference(plumbing.NewTagReferenceName(ref), true); err == nil {
		return r.Hash(), nil
	}
	// Try as hash
	h := plumbing.NewHash(ref)
	if h.IsZero() {
		return plumbing.Hash{}, fmt.Errorf("unknown ref: %s", ref)
	}
	return h, nil
}

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

func normalizeToMap(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, v2 := range t {
			m[fmt.Sprint(k)] = v2
		}
		return m, true
	default:
		return nil, false
	}
}

func discoverFiles(fs billy.Filesystem, root string, include, ignore []string) ([]string, error) {
	var out []string
	err := walk(fs, root, func(p string, isDir bool) error {
		if isDir {
			// Check if directory is ignored as a whole
			for _, ig := range ignore {
				if matchGlob(ig, p+"/") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		for _, ig := range ignore {
			if matchGlob(ig, p) {
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
	// Normalize start
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
	// DFS
	var stack []string
	stack = append(stack, start)
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
	// Support ** by translating to path.Match compatible segments.
	// Use filepath.Match after normalizing to slash.
	p := filepath.ToSlash(pattern)
	n := filepath.ToSlash(strings.TrimPrefix(name, "/"))
	// Implement a simple ** support by replacing with a placeholder
	if strings.Contains(p, "**") {
		parts := strings.Split(p, "**")
		if len(parts) == 2 {
			// prefix**suffix -> prefix*suffix with more permissive check
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(n, strings.TrimSuffix(prefix, "/")) && strings.HasSuffix(n, strings.TrimPrefix(suffix, "/"))
		}
	}
	ok, _ := pathMatch(p, n)
	return ok
}

func pathMatch(pattern, name string) (bool, error) {
	// filepath.Match treats path separators specially; we already normalized to forward slashes.
	return filepath.Match(pattern, name)
}

func uniq(ss []string) []string {
	m := make(map[string]struct{}, len(ss))
	var out []string
	for _, s := range ss {
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ------------------------------- Validation -------------------------------- //

type DocWarning struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

func basicValidate(mem *InMemoryConfig) []DocWarning {
	var out []DocWarning
	for _, d := range mem.AllDocs {
		// If it looks like a k8s doc, check apiVersion/kind
		if _, ok := d.Content["kind"]; ok {
			if d.APIVersion() == "" {
				out = append(out, DocWarning{Location: loc(d), Message: "Missing apiVersion"})
			}
			if d.Kind() == "" {
				out = append(out, DocWarning{Location: loc(d), Message: "Missing kind"})
			}
		}
		if d.Kind() == "ConfigMap" {
			_, ok := d.Content["data"].(map[string]any) // todo removed data here
			if !ok {
				out = append(out, DocWarning{Location: loc(d), Message: "ConfigMap.data must be a mapping"})
			}
			meta, _ := d.Content["metadata"].(map[string]any)
			if meta == nil || meta["name"] == nil {
				out = append(out, DocWarning{Location: loc(d), Message: "ConfigMap.metadata.name is required"})
			}
		}
	}
	return out
}

func loc(d LoadedDoc) string { return fmt.Sprintf("%s#doc%d", d.SourcePath, d.IndexInFile) }

// ------------------------------ Summarization ------------------------------- //

type Summary struct {
	TotalDocs int                 `json:"total_docs"`
	ByKind    map[string]int      `json:"by_kind"`
	ByFile    map[string]int      `json:"by_file"`
	Samples   map[string][]string `json:"configmap_samples"`
}

func summarize(mem *InMemoryConfig) Summary {
	byKind := make(map[string]int)
	for k, v := range mem.ByKind {
		byKind[k] = len(v)
	}
	byFile := make(map[string]int)
	for f, v := range mem.ByFile {
		byFile[f] = len(v)
	}
	samples := map[string][]string{}
	if cms, ok := mem.ByKind["ConfigMap"]; ok {
		for i, d := range cms {
			if i >= 5 {
				break
			}
			name := "<unnamed>"
			if meta, _ := d.Content["metadata"].(map[string]any); meta != nil {
				if n, _ := meta["name"].(string); n != "" {
					name = n
				}
			}
			if data, _ := d.Content["data"].(map[string]any); data != nil {
				keys := make([]string, 0, len(data))
				for k := range data {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				samples[name] = keys
			}
		}
	}
	return Summary{TotalDocs: len(mem.AllDocs), ByKind: byKind, ByFile: byFile, Samples: samples}
}

// ------------------------------ Utilities ---------------------------------- //

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(s string) error { *m = append(*m, s); return nil }

// stdoutWriter wraps Stdout but keeps this file standalone for playgrounds.
// In real apps, just use os.Stdout directly.

type stdoutWriter struct{}

func (stdoutWriter) Write(p []byte) (int, error) { fmt.Print(string(p)); return len(p), nil }
