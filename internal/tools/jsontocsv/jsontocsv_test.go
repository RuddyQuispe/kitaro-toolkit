package jsontocsv

import "testing"

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "array of flat objects (header sorted alphabetically)",
			input: `[{"name":"Ana","age":30},{"name":"Beto","age":25}]`,
			want:  "age,name\n30,Ana\n25,Beto\n",
		},
		{
			name:  "nested objects flattened with dot notation",
			input: `[{"user":{"name":"Ana","address":{"city":"BA"}}}]`,
			want:  "user.address.city,user.name\nBA,Ana\n",
		},
		{
			name:  "rows with differing keys get union header, blanks for missing",
			input: `[{"a":1,"b":2},{"a":3,"c":4}]`,
			want:  "a,b,c\n1,2,\n3,,4\n",
		},
		{
			name:  "single object becomes a one-row table",
			input: `{"a":1,"b":2}`,
			want:  "a,b\n1,2\n",
		},
		{
			name:  "values needing quotes are quoted per RFC 4180",
			input: `[{"note":"has, a comma"}]`,
			want:  "note\n\"has, a comma\"\n",
		},
		{
			name:    "malformed json errors",
			input:   `{"a":}`,
			wantErr: true,
		},
		{
			name:    "empty array errors",
			input:   `[]`,
			wantErr: true,
		},
		{
			name:    "empty input errors",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.input, nil)
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
