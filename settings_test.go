package main

import "testing"

func TestHistoryKeepDays(t *testing.T) {
	cases := map[string]int{
		"30 天":  30,
		"90 天":  90,
		"180 天": 180,
		"365 天": 365,
		"永久":    0,
		"未知":    0,
	}
	for value, want := range cases {
		if got := historyKeepDays(value); got != want {
			t.Errorf("historyKeepDays(%q) = %d, want %d", value, got, want)
		}
	}
}
