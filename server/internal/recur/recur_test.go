package recur

import "testing"

func TestNext(t *testing.T) {
	const (
		mon = 1 << iota
		tue
		wed
		thu
		fri
		sat
		sun
	)

	tests := []struct {
		name   string
		rule   Rule
		oldDue string
		doneOn string
		today  string
		want   string
		wantOK bool
	}{
		// after_done counts from the day it was done.
		{"after_done daily", Rule{Freq: "day", Every: 3, Mode: "after_done"}, "2026-03-01", "2026-03-05", "2026-03-05", "2026-03-08", true},
		{"after_done weekly", Rule{Freq: "week", Every: 2, Mode: "after_done"}, "", "2026-03-05", "2026-03-05", "2026-03-19", true},
		{"after_done monthly", Rule{Freq: "month", Every: 1, Mode: "after_done"}, "", "2026-03-15", "2026-03-15", "2026-04-15", true},
		{"after_done month end clamps", Rule{Freq: "month", Every: 1, Mode: "after_done"}, "", "2026-01-31", "2026-01-31", "2026-02-28", true},
		{"after_done leap year", Rule{Freq: "month", Every: 1, Mode: "after_done"}, "", "2028-01-31", "2028-01-31", "2028-02-29", true},
		{"after_done across year", Rule{Freq: "day", Every: 1, Mode: "after_done"}, "", "2026-12-31", "2026-12-31", "2027-01-01", true},

		// fixed counts from the old due date, whenever it was done.
		{"fixed daily", Rule{Freq: "day", Every: 1, Mode: "fixed"}, "2026-03-01", "2026-03-04", "2026-03-04", "2026-03-02", true},
		{"fixed without old due uses done day", Rule{Freq: "day", Every: 2, Mode: "fixed"}, "", "2026-03-04", "2026-03-04", "2026-03-06", true},
		{"fixed weekly no mask", Rule{Freq: "week", Every: 1, Mode: "fixed"}, "2026-03-02", "2026-03-02", "2026-03-02", "2026-03-09", true},
		// 2026-03-02 is a Monday.
		{"fixed mask same week", Rule{Freq: "week", Every: 1, Weekdays: mon | thu, Mode: "fixed"}, "2026-03-02", "2026-03-02", "2026-03-02", "2026-03-05", true},
		{"fixed mask next week", Rule{Freq: "week", Every: 1, Weekdays: mon | thu, Mode: "fixed"}, "2026-03-05", "2026-03-05", "2026-03-05", "2026-03-09", true},
		{"fixed mask every 2 weeks skips one", Rule{Freq: "week", Every: 2, Weekdays: mon | thu, Mode: "fixed"}, "2026-03-05", "2026-03-05", "2026-03-05", "2026-03-16", true},
		{"fixed mask sunday ends week", Rule{Freq: "week", Every: 1, Weekdays: sun, Mode: "fixed"}, "2026-03-08", "2026-03-08", "2026-03-08", "2026-03-15", true},
		{"fixed mask all days", Rule{Freq: "week", Every: 1, Weekdays: mon | tue | wed | thu | fri | sat | sun, Mode: "fixed"}, "2026-03-08", "2026-03-08", "2026-03-08", "2026-03-09", true},
		{"fixed monthly", Rule{Freq: "month", Every: 1, Mode: "fixed"}, "2026-03-15", "2026-03-20", "2026-03-20", "2026-04-15", true},
		{"fixed monthly 31st to February", Rule{Freq: "month", Every: 1, Mode: "fixed"}, "2026-01-31", "2026-01-31", "2026-01-31", "2026-02-28", true},
		{"fixed monthly end of February back to 31st", Rule{Freq: "month", Every: 1, Mode: "fixed"}, "2026-02-28", "2026-02-28", "2026-02-28", "2026-03-31", true},
		{"fixed monthly 30th clamps", Rule{Freq: "month", Every: 1, Mode: "fixed"}, "2026-01-30", "2026-01-30", "2026-01-30", "2026-02-28", true},
		{"fixed quarterly", Rule{Freq: "month", Every: 3, Mode: "fixed"}, "2026-11-30", "2026-11-30", "2026-11-30", "2027-02-28", true},

		// until
		{"until allows last", Rule{Freq: "day", Every: 1, Mode: "fixed", Until: "2026-03-02"}, "2026-03-01", "2026-03-01", "2026-03-01", "2026-03-02", true},
		{"until passed by next", Rule{Freq: "day", Every: 1, Mode: "fixed", Until: "2026-03-01"}, "2026-03-01", "2026-03-01", "2026-03-01", "", false},
		{"until already in the past", Rule{Freq: "day", Every: 1, Mode: "fixed", Until: "2026-03-10"}, "2026-03-01", "2026-03-12", "2026-03-12", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := Next(tt.rule, tt.oldDue, tt.doneOn, tt.today)
			if err != nil {
				t.Fatalf("Next() error: %v", err)
			}
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Next() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNextErrors(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		done string
	}{
		{"every zero", Rule{Freq: "day", Every: 0, Mode: "fixed"}, "2026-03-01"},
		{"bad mode", Rule{Freq: "day", Every: 1, Mode: "sometimes"}, "2026-03-01"},
		{"bad freq", Rule{Freq: "year", Every: 1, Mode: "fixed"}, "2026-03-01"},
		{"bad date", Rule{Freq: "day", Every: 1, Mode: "after_done"}, "2026-02-30"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := Next(tt.rule, "", tt.done, tt.done); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"2026-02-28", true},
		{"2026-02-29", false},
		{"2028-02-29", true},
		{"2026-2-28", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.in); got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
