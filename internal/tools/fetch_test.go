package tools

import "testing"

func TestIsOfflineAllowedHost(t *testing.T) {
	allowed := []string{
		"localhost",
		"127.0.0.1",
		"10.10.10.5",
		"172.16.1.2",
		"192.168.56.101",
		"fe80::1",
		"challenge.local",
		"target",
	}
	for _, host := range allowed {
		if !isOfflineAllowedHost(host) {
			t.Fatalf("expected %q to be allowed in offline mode", host)
		}
	}

	blocked := []string{
		"example.com",
		"8.8.8.8",
		"1.1.1.1",
	}
	for _, host := range blocked {
		if isOfflineAllowedHost(host) {
			t.Fatalf("expected %q to be blocked in offline mode", host)
		}
	}
}

func TestEnsureOfflineURLRejectsUnsafeForms(t *testing.T) {
	blocked := []string{
		"ftp://127.0.0.1/file",
		"http://user:pass@127.0.0.1/",
		"http://127.0.0.1:99999/",
		"http:///missing-host",
		"http://example.com/",
	}
	for _, rawURL := range blocked {
		if err := ensureOfflineURL(rawURL); err == nil {
			t.Fatalf("expected %q to be rejected in offline mode", rawURL)
		}
	}
}

func TestEnsureOfflineURLAllowsLocalTargets(t *testing.T) {
	allowed := []string{
		"http://localhost:8080/",
		"https://127.0.0.1/",
		"http://192.168.56.101/",
		"http://target/path",
		"http://challenge.local/",
	}
	for _, rawURL := range allowed {
		if err := ensureOfflineURL(rawURL); err != nil {
			t.Fatalf("expected %q to be allowed in offline mode: %v", rawURL, err)
		}
	}
}
