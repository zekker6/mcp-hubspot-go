package main

import "testing"

func TestResolveReadOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagValue bool
		flagSet   bool
		envValue  string
		want      bool
	}{
		{
			name:      "flag explicitly true wins over empty env",
			flagValue: true,
			flagSet:   true,
			envValue:  "",
			want:      true,
		},
		{
			name:      "flag explicitly false wins over env true",
			flagValue: false,
			flagSet:   true,
			envValue:  "true",
			want:      false,
		},
		{
			name:      "flag explicitly true wins over env false",
			flagValue: true,
			flagSet:   true,
			envValue:  "false",
			want:      true,
		},
		{
			name:      "flag at default, env true",
			flagValue: false,
			flagSet:   false,
			envValue:  "true",
			want:      true,
		},
		{
			name:      "flag at default, env false",
			flagValue: false,
			flagSet:   false,
			envValue:  "false",
			want:      false,
		},
		{
			name:      "flag at default, env empty",
			flagValue: false,
			flagSet:   false,
			envValue:  "",
			want:      false,
		},
		{
			name:      "flag at default, env garbage resolves to false",
			flagValue: false,
			flagSet:   false,
			envValue:  "not-a-bool",
			want:      false,
		},
		{
			name:      "flag at default, env 1 resolves to true",
			flagValue: false,
			flagSet:   false,
			envValue:  "1",
			want:      true,
		},
		{
			name:      "flag at default, env 0 resolves to false",
			flagValue: false,
			flagSet:   false,
			envValue:  "0",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveReadOnly(tt.flagValue, tt.flagSet, tt.envValue)
			if got != tt.want {
				t.Fatalf("resolveReadOnly(%v, %v, %q) = %v, want %v",
					tt.flagValue, tt.flagSet, tt.envValue, got, tt.want)
			}
		})
	}
}
