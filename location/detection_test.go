package location

import (
	"path/filepath"
	"testing"
)

func TestDetectCountryWithDatabase(t *testing.T) {
	tests := []struct {
		ip      string
		want    string
		wantErr bool
	}{
		{"8.8.8.8", "US", false},
		{"8.8.4.4", "US", false},
		{"8.8.8.8.8", "", true},
		{"127.0.0.1", "", true},
		{"asd", "", true},
	}
	for _, tt := range tests {
		got, err := DetectCountryWithDatabase(tt.ip, filepath.Join("../", Database))
		if (err != nil) != tt.wantErr {
			t.Errorf("DetectCountry() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("DetectCountry() = %v, want %v", got, tt.want)
		}
	}
}
