package cmd

import (
	"errors"
	"testing"
)

func TestPeriodMonths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		years   int
		want    int
		wantErr bool
	}{
		{years: 1, want: 12},
		{years: 2, want: 24},
		{years: 10, want: 120},
		{years: 0, wantErr: true},
		{years: -1, wantErr: true},
		// Guards against the old workaround of passing months (--period=24).
		{years: 24, wantErr: true},
	}

	for _, tt := range tests {
		got, err := periodMonths(tt.years)
		if tt.wantErr {
			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != CodeUsage {
				t.Errorf("periodMonths(%d) error = %v, want usage ExitError", tt.years, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("periodMonths(%d) = %d, %v; want %d", tt.years, got, err, tt.want)
		}
	}
}

func TestParserAcceptsCheckTLDsFlag(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--tlds", "--tl-ds", "-t"} {
		parser, err := newParser()
		if err != nil {
			t.Fatalf("newParser() error = %v", err)
		}
		if _, err := parser.Parse([]string{"domain", "check", "nimbu", flag, "dev,link"}); err != nil {
			t.Errorf("Parse(%s) error = %v", flag, err)
		}
	}
}
