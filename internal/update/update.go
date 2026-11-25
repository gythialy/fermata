// Package update asks GitHub for the app's newest release.
package update

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Endpoint is the release the app watches.
const Endpoint = "https://api.github.com/repos/gythialy/fermata/releases/latest"

// Latest returns the newest release's version and page when it is newer than
// current, and two empty strings when it is not. Errors are the network's own,
// and never stop the app.
func Latest(current string) (version, url string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, Endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Fermata/"+current)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}
	var release struct {
		Tag    string `json:"tag_name"`
		URL    string `json:"html_url"`
		Draft  bool   `json:"draft"`
		Prerel bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", "", err
	}
	if release.Draft || release.Prerel || release.Tag == "" {
		return "", "", nil
	}
	if !Newer(release.Tag, current) {
		return "", "", nil
	}
	return strings.TrimPrefix(release.Tag, "v"), release.URL, nil
}

// Newer reports whether version a is newer than b, comparing dotted numbers
// and ignoring a leading v.
func Newer(a, b string) bool {
	as, bs := parts(a), parts(b)
	for i := 0; i < max(len(as), len(bs)); i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

func parts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v = strings.SplitN(v, "-", 2)[0]
	var out []int
	for _, field := range strings.Split(v, ".") {
		n, err := strconv.Atoi(field)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
