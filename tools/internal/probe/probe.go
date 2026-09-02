// Package probe is the small amount of shared machinery the probes under tools/
// need: reading .env, authenticating, and issuing read-only requests.
//
// It is standard library only, deliberately. A probe has to be runnable before
// an environment has been built.
package probe

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// The client identification AuthenticateByName requires. Either spelling of the
// header is accepted; this sends the Authorization one, which is what the video
// client sends and what behaviours 2.4 measured.
const clientGrammar = `MediaBrowser Client="Atrium probe", Device="atrium-go", ` +
	`DeviceId="atrium-go-probe", Version="0.0.0"`

// Session is an authenticated connection to a reference server.
type Session struct {
	BaseURL string
	Token   string
	http    *http.Client
}

// Flags registers the flags every probe takes and returns a function that opens
// the session. The URL flag exists because the path a measurement travelled is
// part of what it measured: chunked framing and header spelling are visible over
// plain HTTP/1.1 and are not visible over HTTP/2.
func Flags() func() (*Session, error) {
	url := flag.String("url", "", "reference server; defaults to JELLYFIN_URL from .env")
	envPath := flag.String("env", ".env", "path to the credentials file")
	return func() (*Session, error) {
		env, err := readEnv(*envPath)
		if err != nil {
			return nil, err
		}
		base := *url
		if base == "" {
			base = env["JELLYFIN_URL"]
		}
		if base == "" {
			return nil, errors.New("no server: pass -url or set JELLYFIN_URL in .env")
		}
		return open(strings.TrimRight(base, "/"), env)
	}
}

func readEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w (copy .env.example to .env)", path, err)
	}
	defer f.Close()
	env := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return env, sc.Err()
}

func open(base string, env map[string]string) (*Session, error) {
	s := &Session{BaseURL: base, http: &http.Client{Timeout: 120 * time.Second}}
	if tok := env["JELLYFIN_TOKEN"]; tok != "" {
		s.Token = tok
		return s, nil
	}
	user, pw := env["JELLYFIN_USERNAME"], env["JELLYFIN_PASSWORD"]
	if user == "" || pw == "" {
		return nil, errors.New("no JELLYFIN_TOKEN and no JELLYFIN_USERNAME/JELLYFIN_PASSWORD")
	}
	body, _ := json.Marshal(map[string]string{"Username": user, "Pw": pw})
	req, _ := http.NewRequest(http.MethodPost, base+"/Users/AuthenticateByName", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", clientGrammar)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AuthenticateByName answered %d", resp.StatusCode)
	}
	var out struct{ AccessToken string }
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, errors.New("AuthenticateByName returned no AccessToken")
	}
	s.Token = out.AccessToken
	return s, nil
}

// Get issues a read-only request and returns the raw body. Probes read bytes,
// never a re-serialisation of them (Principle VIII).
func (s *Session) Get(path string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, s.BaseURL+path, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", `MediaBrowser Token="`+s.Token+`", `+
		strings.TrimPrefix(clientGrammar, "MediaBrowser "))
	resp, err := s.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

// Walk visits every (key path, value) pair of a decoded JSON document. Numbers
// arrive as json.Number so their literal text survives.
func Walk(v any, key string, fn func(key string, val any)) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			fn(k, child)
			Walk(child, k, fn)
		}
	case []any:
		for _, child := range t {
			Walk(child, key, fn)
		}
	}
}

// Decode parses a body with number literals preserved.
func Decode(body []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}
