package azaks

import "testing"

func TestEnsureOutputFormatArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty args",
			input: "",
			want:  "--output json",
		},
		{
			name:  "already has long output flag",
			input: "--resource-group rg --output table",
			want:  "--resource-group rg --output table",
		},
		{
			name:  "already has long output flag equals",
			input: "--output=json --resource-group rg",
			want:  "--output=json --resource-group rg",
		},
		{
			name:  "already has short output flag",
			input: "-o json --name test",
			want:  "-o json --name test",
		},
		{
			name:  "already has short compact output flag",
			input: "-ojson --name test",
			want:  "-ojson --name test",
		},
		{
			name:  "no output flag appends json",
			input: "--name test --resource-group rg",
			want:  "--name test --resource-group rg --output json",
		},
		{
			name:    "malformed args returns error",
			input:   "--name \"missing-end",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ensureOutputFormatArgs(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
