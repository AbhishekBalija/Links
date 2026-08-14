package config

import "testing"

func TestFrontendURL(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "local dev — no VERCEL vars",
			env:  map[string]string{},
			want: "http://localhost:5173",
		},
		{
			name: "production",
			env: map[string]string{
				"VERCEL_ENV":                    "production",
				"VERCEL_PROJECT_PRODUCTION_URL": "links-campus.vercel.app",
			},
			want: "https://links-campus.vercel.app",
		},
		{
			name: "preview",
			env: map[string]string{
				"VERCEL_ENV": "preview",
				"VERCEL_URL": "links-git-feat-x.vercel.app",
			},
			want: "https://links-git-feat-x.vercel.app",
		},
		{
			name: "explicit override wins",
			env: map[string]string{
				"FRONTEND_URL":                  "https://custom.example.com",
				"VERCEL_ENV":                    "production",
				"VERCEL_PROJECT_PRODUCTION_URL": "links-campus.vercel.app",
			},
			want: "https://custom.example.com",
		},
		{
			name: "VERCEL_URL fallback when production URL absent",
			env: map[string]string{
				"VERCEL_ENV": "production",
				"VERCEL_URL": "links-git-fallback.vercel.app",
			},
			want: "https://links-git-fallback.vercel.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"FRONTEND_URL", "VERCEL_ENV", "VERCEL_PROJECT_PRODUCTION_URL", "VERCEL_URL"} {
				t.Setenv(key, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got := frontendURL()
			if got != tt.want {
				t.Errorf("frontendURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
