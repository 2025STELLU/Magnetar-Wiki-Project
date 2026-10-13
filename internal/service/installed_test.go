package service

import (
	"strings"
	"testing"
)

func TestSystemdFirstReadsWhatSystemdQuoteWrites(t *testing.T) {
	for _, path := range []string{"/opt/magnetar/magnetar", "/www/wwwroot/my wiki/magnetar", `/srv/50%$off/"q"\x/magnetar`, "/srv/维基/magnetar"} {
		first, rest := systemdFirst(systemdQuote(path) + ` "update" "-auto"`)
		if first != path {
			t.Errorf("systemdFirst(systemdQuote(%q)) = %q, want %q", path, first, path)
		}
		if rest != `"update" "-auto"` {
			t.Errorf("systemdFirst(systemdQuote(%q)) rest = %q, want the arguments", path, rest)
		}
	}
}

func TestPointSystemdAt(t *testing.T) {
	s := Spec{Name: "magnetar", Executable: "/www/wwwroot/magnetar/magnetar", Root: "/www/wwwroot/magnetar",
		UpdateArgs: []string{"update", "-auto", "-service", "magnetar", "-data-dir", "/www/wwwroot/magnetar", "--"}}
	unit, _ := s.SystemdUpdate()
	next, changed := pointSystemdAt(unit, "/var/lib/magnetar/abc/bin/magnetar")
	if !changed {
		t.Fatal("pointSystemdAt() changed = false, want true")
	}
	s.UpdateExecutable = "/var/lib/magnetar/abc/bin/magnetar"
	want, _ := s.SystemdUpdate()
	if next != want {
		t.Errorf("pointSystemdAt() = %q, want %q", next, want)
	}
	if _, again := pointSystemdAt(next, "/var/lib/magnetar/abc/bin/magnetar"); again {
		t.Error("pointSystemdAt(already pointed) changed = true, want false")
	}
	if got := systemdProgram(next); got != "/var/lib/magnetar/abc/bin/magnetar" {
		t.Errorf("systemdProgram() = %q, want the new program", got)
	}
}

func TestPointPlistAt(t *testing.T) {
	s := Spec{Name: "magnetar", Executable: "/Users/a/magnetar & co/magnetar", Root: "/Users/a/magnetar",
		UpdateArgs: []string{"update", "-auto"}}
	plist := s.LaunchdUpdate()
	if got := plistFirst(plist); got != s.Executable {
		t.Errorf("plistFirst() = %q, want %q", got, s.Executable)
	}
	next, changed := pointPlistAt(plist, "/Library/Application Support/magnetar/x/bin/magnetar")
	if !changed {
		t.Fatal("pointPlistAt() changed = false, want true")
	}
	s.UpdateExecutable = "/Library/Application Support/magnetar/x/bin/magnetar"
	if want := s.LaunchdUpdate(); next != want {
		t.Errorf("pointPlistAt() = %q, want %q", next, want)
	}
	if !strings.Contains(next, "<string>update</string>") {
		t.Errorf("pointPlistAt() = %q, want the arguments kept", next)
	}
}

func TestSystemdUser(t *testing.T) {
	s := Spec{Name: "magnetar", Executable: "/opt/magnetar/magnetar", Root: "/opt/magnetar", User: "www"}
	if got, _ := systemdValue(s.Systemd(), "User"); got != "www" {
		t.Errorf("systemdValue(User) = %q, want %q", got, "www")
	}
	s.User = ""
	if _, ok := systemdValue(s.Systemd(), "User"); ok {
		t.Error("systemdValue(User) ok = true for a unit without one, want false")
	}
}
