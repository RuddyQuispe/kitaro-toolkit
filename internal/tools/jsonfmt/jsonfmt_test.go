package jsonfmt

import "testing"

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		opts    map[string]string
		want    string
		wantErr bool
	}{
		{
			name:  "pretty print default",
			input: `{"b":1,"a":2}`,
			opts:  nil,
			want:  "{\n  \"b\": 1,\n  \"a\": 2\n}",
		},
		{
			name:  "pretty print explicit mode",
			input: `{"a":[1,2,3]}`,
			opts:  map[string]string{"mode": "pretty"},
			want:  "{\n  \"a\": [\n    1,\n    2,\n    3\n  ]\n}",
		},
		{
			name:  "minify",
			input: "{\n  \"a\": 1,\n  \"b\": 2\n}",
			opts:  map[string]string{"mode": "minify"},
			want:  `{"a":1,"b":2}`,
		},
		{
			name:  "preserves key order",
			input: `{"z":1,"a":2,"m":3}`,
			opts:  nil,
			want:  "{\n  \"z\": 1,\n  \"a\": 2,\n  \"m\": 3\n}",
		},
		{
			name:    "malformed json returns error",
			input:   `{"a":}`,
			wantErr: true,
		},
		{
			name:    "empty input returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:  "unicode preserved",
			input: `{"name":"José"}`,
			opts:  nil,
			want:  "{\n  \"name\": \"José\"\n}",
		},
		{
			name:  "large numbers preserved verbatim",
			input: `{"n":123456789012345678}`,
			opts:  nil,
			want:  "{\n  \"n\": 123456789012345678\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.input, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (output: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
