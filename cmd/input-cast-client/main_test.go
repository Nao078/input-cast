package main

import "testing"

func TestNormalizeServerHost(t *testing.T) {
	tests := map[string]string{
		"":                                       "",
		"localhost":                              "localhost:8080",
		"localhost:9090":                         "localhost:9090",
		"192.168.0.10":                           "192.168.0.10:8080",
		"192.168.0.10:9090":                      "192.168.0.10:9090",
		"input-cast.local":                       "input-cast.local:8080",
		"http://input-cast.local:9090/foo/bar":   "input-cast.local:9090",
		"https://192.168.0.10/api/input/gamepad": "192.168.0.10:8080",
	}

	for input, want := range tests {
		if got := normalizeServerHost(input); got != want {
			t.Fatalf("normalizeServerHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestServerURLForSettings(t *testing.T) {
	if got := serverURLForSettings(false, "example.test:9090"); got != defaultServerURL {
		t.Fatalf("serverURLForSettings(false, custom) = %q, want %q", got, defaultServerURL)
	}

	want := "http://192.168.0.10:8080/api/input/gamepad"
	if got := serverURLForSettings(true, "192.168.0.10"); got != want {
		t.Fatalf("serverURLForSettings(true, host) = %q, want %q", got, want)
	}
}
