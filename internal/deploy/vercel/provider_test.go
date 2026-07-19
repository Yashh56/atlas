package vercel

import (
	"testing"
)

func TestParseVercelURL(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantURL string
		wantErr bool
	}{
		{
			name: "realistic success",
			output: `Vercel CLI 34.0.0
Inspect: https://vercel.com/yash/todo-app/1234
Production: https://todo-app-abc123.vercel.app
`,
			wantURL: "https://todo-app-abc123.vercel.app",
			wantErr: false,
		},
		{
			name: "garbage output",
			output: `Deploying...
Error: Something went wrong
`,
			wantURL: "",
			wantErr: true,
		},
		{
			name: "multiple urls uses last",
			output: `
https://old.vercel.app
https://new.vercel.app
`,
			wantURL: "https://new.vercel.app",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVercelURL(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVercelURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantURL {
				t.Errorf("parseVercelURL() = %v, want %v", got, tt.wantURL)
			}
		})
	}
}
