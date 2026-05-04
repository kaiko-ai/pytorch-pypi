// Command pytorch-pypi mirrors the PyTorch download.pytorch.org index trees
// into a PEP 503 simple/ layout suitable for serving via GitHub Pages.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	stdhtml "html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/html"
)

const (
	baseURL    = "https://download.pytorch.org"
	minRootPkg = 40
)

var (
	safeNameRE  = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)
	timestampRE = regexp.MustCompile(`(?m)^.*TIMESTAMP 1.*\r?\n?`)
	rewriteFrom = []byte(`href="/whl`)
	rewriteTo   = []byte(`href="https://download.pytorch.org/whl`)
	whlURLBytes = []byte("https://download.pytorch.org/whl/")
)

// Compute-platform variants. Retired ones are kept here as reference:
//
//	cu75 cu80 cu90 cu91 cu92 cu100 cu101 cu102 cu110 cu111 cu113 cu115 cu116 cu117 cu117_pypi_cudnn
//	rocm3.10 rocm3.7 rocm3.8 rocm4.0.1 rocm4.1 rocm4.2 rocm4.3.1 rocm4.5.2 rocm5.0 rocm5.1.1 rocm5.2 rocm5.3 rocm5.4.2 rocm5.5 rocm5.6 rocm5.7
var variants = []string{
	"cpu", "cpu-cxx11-abi", "cpu_pypi_pkg",
	"cu118", "cu121", "cu124", "cu126", "cu128", "cu129", "cu130",
	"rocm6.0", "rocm6.1", "rocm6.2", "rocm6.2.4", "rocm6.3", "rocm6.4",
	"xpu",
}

func isSafeName(s string) bool {
	switch s {
	case "", ".", "..":
		return false
	}
	if strings.ContainsAny(s, "/") || strings.Contains(s, "..") {
		return false
	}
	return safeNameRE.MatchString(s)
}

type fetcher struct {
	client *http.Client
}

func (f *fetcher) get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func extractAnchorTexts(b []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var names []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					name := strings.TrimSpace(c.Data)
					name = strings.TrimSuffix(name, "/")
					if name != "" {
						names = append(names, name)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return names, nil
}

func countLinesContaining(b, needle []byte) int {
	n := 0
	for _, line := range bytes.Split(b, []byte("\n")) {
		if bytes.Contains(line, needle) {
			n++
		}
	}
	return n
}

func updateIndex(ctx context.Context, f *fetcher, d string, workers int) error {
	simpleDir := filepath.Join(d, "simple")
	if err := os.MkdirAll(simpleDir, 0o755); err != nil {
		return err
	}

	rootURL := baseURL + "/" + d + "/"
	body, status, err := f.get(ctx, rootURL)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", d, err)
	}
	if status != 200 {
		return fmt.Errorf("fetch %s: HTTP %d", d, status)
	}
	body = timestampRE.ReplaceAll(body, nil)
	if err := os.WriteFile(filepath.Join(simpleDir, "index.html"), body, 0o644); err != nil {
		return err
	}

	names, err := extractAnchorTexts(body)
	if err != nil {
		return fmt.Errorf("parse %s index: %w", d, err)
	}

	fmt.Printf("%s => %s/simple/\n", rootURL, d)

	if len(names) < minRootPkg {
		return fmt.Errorf("low package count for %s: %d (probably intermittent download failure)", d, len(names))
	}

	type job struct{ name string }
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var done int32
	total := len(names)

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			p := j.name
			if !isSafeName(p) {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "skipping unsafe name: %s\n", p)
				mu.Unlock()
				continue
			}
			projDir := filepath.Join(simpleDir, p)
			if err := os.MkdirAll(projDir, 0o755); err != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", projDir, err)
				mu.Unlock()
				continue
			}
			outPath := filepath.Join(projDir, "index.html")
			projURL := fmt.Sprintf("%s/%s/%s/", baseURL, d, p)
			pBody, pStatus, ferr := f.get(ctx, projURL)
			if ferr != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "skipping %s/%s: %v\n", d, p, ferr)
				mu.Unlock()
				_ = os.WriteFile(outPath, nil, 0o644)
				continue
			}
			if pStatus != 200 {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "skipping %s/%s: HTTP %d\n", d, p, pStatus)
				mu.Unlock()
				_ = os.WriteFile(outPath, nil, 0o644)
				continue
			}
			pBody = bytes.ReplaceAll(pBody, rewriteFrom, rewriteTo)
			pBody = timestampRE.ReplaceAll(pBody, nil)
			if err := os.WriteFile(outPath, pBody, 0o644); err != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "write %s: %v\n", outPath, err)
				mu.Unlock()
				continue
			}
			pcount := countLinesContaining(pBody, whlURLBytes)
			i := atomic.AddInt32(&done, 1)
			mu.Lock()
			fmt.Printf("%5d / %d                   %s/%s/ => %s/simple/%s/ %d\n",
				i, total, d, p, d, p, pcount)
			mu.Unlock()
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}
	for _, n := range names {
		jobs <- job{name: n}
	}
	close(jobs)
	wg.Wait()
	fmt.Println()
	return nil
}

const rootIndexHeader = `<!DOCTYPE html>
<html>
  <body>
    <h1>Sonatype <a href="https://pytorch.org/">PyTorch</a> PyPI improved indexes for <a href="https://help.sonatype.com/en/pypi-repositories.html#download--search--and-install-packages-using-pip">Nexus Repository</a></h1>

Generated by <a href="https://github.com/sonatype-nexus-community/pytorch-pypi">sonatype-nexus-community/pytorch-pypi</a> from <a href="https://download.pytorch.org/whl/">https://download.pytorch.org/whl/</a>
<p>
for the full index, use <code>--index-url <a href="simple">https://sonatype-nexus-community.github.io/pytorch-pypi/whl/simple</a></code>
<p>
You can also use compute platform filtered indexes:
<ul>
`

const rootIndexFooter = `</ul>
</body>
</html>
`

func updateHumanIndexes() error {
	entries, err := os.ReadDir("whl")
	if err != nil {
		return err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if n == "simple" || !isSafeName(n) {
			continue
		}
		dirs = append(dirs, n)
	}
	sort.Strings(dirs)

	var b bytes.Buffer
	b.WriteString(rootIndexHeader)
	for _, d := range dirs {
		e := stdhtml.EscapeString(d)
		fmt.Fprintf(&b, "<li><a href=\"%s\">%s</a></li>\n", e, e)
	}
	b.WriteString(rootIndexFooter)
	if err := os.WriteFile(filepath.Join("whl", "index.html"), b.Bytes(), 0o644); err != nil {
		return err
	}

	for _, d := range dirs {
		e := stdhtml.EscapeString(d)
		body := fmt.Sprintf(`<!DOCTYPE html>
<html>
  <body>
    <h1>Sonatype <a href="https://pytorch.org/">PyTorch</a> PyPI improved indexes for <a href="https://help.sonatype.com/en/pypi-repositories.html#download--search--and-install-packages-using-pip">Nexus Repository</a></h1>

Generated by <a href="https://github.com/sonatype-nexus-community/pytorch-pypi">sonatype-nexus-community/pytorch-pypi</a> from <a href="https://download.pytorch.org/whl/%s/">https://download.pytorch.org/whl/%s/</a>
<p>
for %s compute platform index, use <code>--index-url <a href="simple">https://sonatype-nexus-community.github.io/pytorch-pypi/whl/%s/simple</a></code>
<p>
see also <a href="..">other available indexes</a>
</body>
</html>
`, e, e, e, e)
		if err := os.WriteFile(filepath.Join("whl", d, "index.html"), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeSummary() error {
	var paths []string
	paths = append(paths, filepath.Join("whl", "simple"))
	matches, _ := filepath.Glob(filepath.Join("whl", "*", "simple"))
	sort.Strings(matches)
	for _, m := range matches {
		if m == filepath.Join("whl", "simple") {
			continue
		}
		paths = append(paths, m)
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "# generated %s\n", time.Now().UTC().Format(time.RFC3339))
	for _, p := range paths {
		entries, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%d %s\n", len(entries), p)
	}
	if err := os.WriteFile("summary.txt", b.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Print(b.String())
	return nil
}

func main() {
	root := flag.String("root", "gh-pages", "directory holding the generated index tree")
	workers := flag.Int("workers", 32, "concurrent HTTP workers")
	timeout := flag.Duration("timeout", 60*time.Second, "per-request HTTP timeout")
	flag.Parse()

	if _, err := os.Stat(*root); err != nil {
		log.Fatalf("root %q missing: %v", *root, err)
	}
	if err := os.Chdir(*root); err != nil {
		log.Fatalf("chdir %q: %v", *root, err)
	}

	ctx := context.Background()
	f := &fetcher{client: &http.Client{Timeout: *timeout}}

	if err := updateIndex(ctx, f, "whl", *workers); err != nil {
		log.Fatal(err)
	}
	if err := updateIndex(ctx, f, "whl/nightly", *workers); err != nil {
		log.Fatal(err)
	}
	for _, v := range variants {
		if err := updateIndex(ctx, f, "whl/"+v, *workers); err != nil {
			log.Fatal(err)
		}
	}
	if err := updateHumanIndexes(); err != nil {
		log.Fatal(err)
	}
	if err := writeSummary(); err != nil {
		log.Fatal(err)
	}
}
