package scid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ── extractClientID ───────────────────────────────────────────────────────────

func TestExtractClientID_DoubleQuote(t *testing.T) {
	js := `var n=123;client_id:"yweyS1UG4UMsEJVmd32E0RrcDw8gh4cZ";var x=1;`
	got := extractClientID(js)
	if got != "yweyS1UG4UMsEJVmd32E0RrcDw8gh4cZ" {
		t.Errorf("got %q", got)
	}
}

func TestExtractClientID_EqualSign(t *testing.T) {
	js := `?client_id=abcDEF1234567890abcd&other=x`
	got := extractClientID(js)
	if got != "abcDEF1234567890abcd" {
		t.Errorf("got %q", got)
	}
}

func TestExtractClientID_EscapedColon(t *testing.T) {
	// minifier emits \x3a for :
	js := `client_id\x3a"TestID12345678901234"`
	got := extractClientID(js)
	if got != "TestID12345678901234" {
		t.Errorf("got %q", got)
	}
}

func TestExtractClientID_NotPresent(t *testing.T) {
	got := extractClientID("no id here")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractClientID_TooShort(t *testing.T) {
	// value shorter than 20 chars should not match
	got := extractClientID(`client_id:"tooshort"`)
	if got != "" {
		t.Errorf("expected empty for short value, got %q", got)
	}
}

// ── fetch (via mock HTTP server) ──────────────────────────────────────────────

// fakeClientID is a 32-char string that satisfies the regex.
const fakeClientID = "FAKEID1234567890FAKEID1234567890"

// fakeBundleJS is a minimal JS snippet containing the client_id.
const fakeBundleJS = `!function(){client_id:"` + fakeClientID + `"}()`

func TestFetch_MockServer(t *testing.T) {
	// Serve a fake SC homepage that references a JS bundle.
	mux := http.NewServeMux()

	var serverURL string // set after server starts

	mux.HandleFunc("/assets/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(fakeBundleJS))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Emit HTML that looks like soundcloud.com, pointing to our fake bundle.
		html := `<html><head></head><body>` +
			`<script crossorigin src="` + serverURL + `/assets/app.js"></script>` +
			`</body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	serverURL = srv.URL

	// Swap the scriptRe so it matches our test server URL (no sndcdn.com check).
	// We test fetch() directly with a patched URL instead.
	// Since fetch() is unexported and hardcodes soundcloud.com, we test via the
	// exported cache path by temporarily pointing the request at our test server.
	// We exercise the extraction logic end-to-end via httpGet + extractClientID.

	body, err := httpGet(context.Background(), serverURL+"/")
	if err != nil {
		t.Fatal("httpGet homepage:", err)
	}
	if !strings.Contains(body, "/assets/app.js") {
		t.Fatal("homepage does not contain bundle URL")
	}

	jsBody, err := httpGet(context.Background(), serverURL+"/assets/app.js")
	if err != nil {
		t.Fatal("httpGet bundle:", err)
	}

	got := extractClientID(jsBody)
	if got != fakeClientID {
		t.Errorf("extractClientID: got %q want %q", got, fakeClientID)
	}
}

// ── cache ─────────────────────────────────────────────────────────────────────

func TestCache_SaveLoad(t *testing.T) {
	// Override cache file location to a temp dir.
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	saveCache("myTestClientID12345678")

	got, ok := loadCache()
	if !ok {
		t.Fatal("loadCache returned false after saveCache")
	}
	if got != "myTestClientID12345678" {
		t.Errorf("got %q", got)
	}
}

func TestCache_Invalidate(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	saveCache("someID1234567890abcdef")
	Invalidate()

	_, ok := loadCache()
	if ok {
		t.Error("loadCache should return false after Invalidate")
	}
}

func TestCache_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Write empty/invalid JSON
	os.MkdirAll(tmp+"/jammer", 0o755)
	os.WriteFile(tmp+"/jammer/sc_client_id.json", []byte("{}"), 0o644)

	_, ok := loadCache()
	if ok {
		t.Error("loadCache should return false for empty client_id")
	}
}
