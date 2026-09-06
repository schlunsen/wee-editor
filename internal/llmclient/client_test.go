package llmclient

import (
	"testing"
)

func TestParseToolCallArguments(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "empty object",
			input:   "{}",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid object",
			input:   `{"name":"test","count":42}`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   `{broken`,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "nested object",
			input:   `{"config":{"key":"val"},"items":[1,2,3]}`,
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolCallArguments(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToolCallArguments(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("ParseToolCallArguments(%q) got %d keys, want %d", tt.input, len(got), tt.wantLen)
			}
		})
	}
}
