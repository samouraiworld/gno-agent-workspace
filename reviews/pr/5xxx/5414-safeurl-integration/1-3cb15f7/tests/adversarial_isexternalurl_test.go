package safeurl

import "testing"

func TestIsExternalURL_Adversarial(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// These should be EXTERNAL (true) but the substring match misclassifies them
		{"path contains gno.land", "https://evil.com/gno.land/malware", true},
		{"subdomain spoof", "https://gno.land.evil.com/", true},
		{"subdomain spoof 2", "https://notgno.land/something", true},
		{"gno.land in query", "https://evil.com?redirect=gno.land", true},
		{"gno.land in fragment", "https://evil.com#gno.land", true},
		{"gno.land in user info", "https://gno.land@evil.com/", true},

		// These should be INTERNAL (false)
		{"exact gno.land", "https://gno.land/r/demo", false},
		{"subdomain of gno.land", "https://test.gno.land/r/demo", false},

		// Edge cases
		{"javascript scheme", "javascript:alert(1)", false},
		{"mailto scheme", "mailto:user@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExternalURL(tt.url)
			if got != tt.expected {
				t.Errorf("IsExternalURL(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}
