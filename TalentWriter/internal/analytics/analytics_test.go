package analytics

import "testing"

func TestNormalizePagePath(t *testing.T) {
	tests := map[string]string{
		"":                                   "/",
		"/post/demo/?utm_source=x#comments": "/post/demo",
		"https://vantalens.com/archive/?q=1": "/archive",
		"post/demo/index.html":              "/post/demo/index.html",
	}
	for input, want := range tests {
		if got := normalizePagePath(input); got != want {
			t.Fatalf("normalizePagePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsTrackablePage(t *testing.T) {
	trackable := []string{"/", "/post/demo", "/archives", "/page/search"}
	for _, path := range trackable {
		if !isTrackablePage(path) {
			t.Fatalf("isTrackablePage(%q) = false, want true", path)
		}
	}

	blocked := []string{"/api/comments", "/platform/backend", "/preview/", "/scss/style.css", "/js/app.js", "/img/a.png"}
	for _, path := range blocked {
		if isTrackablePage(path) {
			t.Fatalf("isTrackablePage(%q) = true, want false", path)
		}
	}
}
