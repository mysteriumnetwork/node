package location

import (
	"testing"
)

func TestDetectorDetectCountry(t *testing.T) {
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
		detector := NewDetector("../bin/server_package/config/GeoLite2-Country.mmdb")
		got, err := detector.DetectCountry(tt.ip)
		if (err != nil) != tt.wantErr {
			t.Errorf("DetectCountry() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("DetectCountry() = %v, want %v", got, tt.want)
		}
	}
}
