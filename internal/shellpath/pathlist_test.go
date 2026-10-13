package shellpath

import "testing"

func TestPathWith(t *testing.T) {
	cases := []struct {
		name  string
		value string
		dir   string
		want  string
	}{
		{"empty", "", `C:\magnetar`, `C:\magnetar`},
		{"appended", `C:\Windows;C:\Tools`, `C:\magnetar`, `C:\Windows;C:\Tools;C:\magnetar`},
		{"already there", `C:\Windows;C:\magnetar`, `C:\magnetar`, `C:\Windows;C:\magnetar`},
		{"already there in another case", `C:\Windows;c:\MAGNETAR\`, `C:\magnetar`, `C:\Windows;c:\MAGNETAR\`},
		{"empty entries dropped", `C:\Windows;;C:\Tools;`, `C:\magnetar`, `C:\Windows;C:\Tools;C:\magnetar`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathWith(tt.value, tt.dir); got != tt.want {
				t.Errorf("pathWith(%q, %q) = %q, want %q", tt.value, tt.dir, got, tt.want)
			}
		})
	}
}

func TestPathWithout(t *testing.T) {
	cases := []struct {
		name  string
		value string
		dir   string
		want  string
	}{
		{"removed", `C:\Windows;C:\old\magnetar;C:\Tools`, `C:\old\magnetar`, `C:\Windows;C:\Tools`},
		{"removed ignoring case and slash", `C:\Windows;c:\OLD\magnetar\`, `C:\old\magnetar`, `C:\Windows`},
		{"not there", `C:\Windows;C:\Tools`, `C:\old\magnetar`, `C:\Windows;C:\Tools`},
		{"only entry", `C:\old\magnetar`, `C:\old\magnetar`, ``},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathWithout(tt.value, tt.dir); got != tt.want {
				t.Errorf("pathWithout(%q, %q) = %q, want %q", tt.value, tt.dir, got, tt.want)
			}
		})
	}
}

func TestPathHasKeepsUnrelatedEntriesApart(t *testing.T) {
	if pathHas(`C:\magnetar-old;C:\Tools`, `C:\magnetar`) {
		t.Errorf("pathHas(%q, %q) = true, want false", `C:\magnetar-old;C:\Tools`, `C:\magnetar`)
	}
}
