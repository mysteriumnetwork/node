package location

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDetectorDetectCountry(t *testing.T) {
	tests := []struct {
		ip      string
		want    string
		wantErr string
	}{
		{"8.8.8.8", "US", ""},
		{"8.8.4.4", "US", ""},
		{"95.85.39.36", "NL", ""},
		{"127.0.0.1", "", ""},
		{"8.8.8.8.8", "", "failed to parse IP"},
		{"asd", "", "failed to parse IP"},
	}

	detector := NewDetector("../bin/server_package/config/GeoLite2-Country.mmdb")
	for _, tt := range tests {
		got, err := detector.DetectCountry(tt.ip)

		assert.Equal(t, tt.want, got, tt.ip)
		if tt.wantErr != "" {
			assert.EqualError(t, err, tt.wantErr, tt.ip)
		} else {
			assert.NoError(t, err, tt.ip)
		}
	}
}
