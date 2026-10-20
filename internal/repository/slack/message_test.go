package slack

import (
	"testing"
	"time"
)

func TestToMrkdwn(t *testing.T) {
	in := "## Related links\n\n\n\n- [Runbook](https://wiki/rb)\n**cpu** > 90 & <rising>"
	want := "*Related links*\n\n- <https://wiki/rb|Runbook>\n*cpu* &gt; 90 &amp; &lt;rising&gt;"

	if got := toMrkdwn(in); got != want {
		t.Fatalf("toMrkdwn:\n got %q\nwant %q", got, want)
	}
}

func TestHumanDuration(t *testing.T) {
	tests := map[time.Duration]string{
		45 * time.Second:               "45s",
		6*time.Minute + 20*time.Second: "6m",
		65 * time.Minute:               "1h 5m",
		51 * time.Hour:                 "2d 3h",
	}
	for d, want := range tests {
		if got := humanDuration(d); got != want {
			t.Errorf("humanDuration(%s) = %q, want %q", d, got, want)
		}
	}
}
