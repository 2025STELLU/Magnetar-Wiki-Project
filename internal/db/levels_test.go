package db

import "testing"

// Pure unit tests: no database needed.

func TestLevelForXPDefaults(t *testing.T) {
	cases := []struct {
		xp    int64
		level int
	}{
		{0, 1}, {99, 1},
		{100, 2}, {299, 2},
		{300, 3}, {599, 3},
		{600, 4}, {999, 4},
		{1000, 5}, {1499, 5},
		{1500, 6}, {2099, 6},
		{2100, 7}, {2799, 7},
		{2800, 8}, {3599, 8},
		{3600, 9}, {4499, 9},
		{4500, 10}, {999999, 10}, // capped at MaxLevel
		{-5, 1},
	}
	for _, c := range cases {
		if got := LevelForXP(c.xp, DefaultLevelThresholds); got != c.level {
			t.Errorf("LevelForXP(%d) = %d, want %d", c.xp, got, c.level)
		}
	}
}

func TestFormatLevel(t *testing.T) {
	if got := FormatLevel("Lv", 3); got != "Lv3" {
		t.Errorf("FormatLevel(Lv, 3) = %q, want %q", got, "Lv3")
	}
	if got := FormatLevel("段", 5); got != "段5" {
		t.Errorf("FormatLevel(段, 5) = %q, want %q", got, "段5")
	}
}

func TestDefaultThresholdsShape(t *testing.T) {
	if len(DefaultLevelThresholds) != MaxLevel {
		t.Fatalf("thresholds length %d, want MaxLevel %d", len(DefaultLevelThresholds), MaxLevel)
	}
	if DefaultLevelThresholds[0] != 0 {
		t.Errorf("level 1 threshold = %d, want 0", DefaultLevelThresholds[0])
	}
	for i := 1; i < MaxLevel; i++ {
		if DefaultLevelThresholds[i] <= DefaultLevelThresholds[i-1] {
			t.Errorf("thresholds not strictly increasing at level %d", i+1)
		}
	}
}
