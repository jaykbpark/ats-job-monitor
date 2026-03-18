package main

import "testing"

func TestParseDeliverNotificationsArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantLimit  int
		wantDBPath string
		wantErr    bool
	}{
		{
			name:       "db path before limit",
			args:       []string{"/tmp/app.sqlite", "--limit", "5"},
			wantLimit:  5,
			wantDBPath: "/tmp/app.sqlite",
		},
		{
			name:       "limit before db path",
			args:       []string{"--limit", "2", "/tmp/app.sqlite"},
			wantLimit:  2,
			wantDBPath: "/tmp/app.sqlite",
		},
		{
			name:    "missing db path",
			args:    []string{"--limit", "2"},
			wantErr: true,
		},
		{
			name:    "unsupported flag",
			args:    []string{"--dry-run", "/tmp/app.sqlite"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotDBPath, err := parseDeliverNotificationsArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseDeliverNotificationsArgs() error = %v", err)
			}

			if gotLimit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", gotLimit, tt.wantLimit)
			}

			if gotDBPath != tt.wantDBPath {
				t.Fatalf("dbPath = %q, want %q", gotDBPath, tt.wantDBPath)
			}
		})
	}
}
