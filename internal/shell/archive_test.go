package shell

import "testing"

func TestIsArchiveFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		// Tar formats
		{"app.tar", true},
		{"app.tar.gz", true},
		{"app.tgz", true},
		{"app.tar.bz2", true},
		{"app.tbz2", true},
		{"app.tbz", true},
		{"app.tb2", true},
		{"app.tar.xz", true},
		{"app.txz", true},
		{"app.tar.lz", true},
		{"app.tlz", true},
		{"app.tar.lzma", true},
		{"app.tar.Z", true},
		{"app.tZ", true},
		{"app.tpz", true},
		{"app.tar.zst", true},
		{"app.tzst", true},
		// Standalone compression
		{"app.gz", true},
		{"app.bz2", true},
		{"app.xz", true},
		{"app.lz", true},
		{"app.lzma", true},
		{"app.Z", true},
		// Non-archive
		{"app.zip", false},
		{"package.json", false},
		{"README.md", false},
		{"app.tar.gz.sig", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsArchiveFilename(tt.name); got != tt.want {
				t.Errorf("IsArchiveFilename(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsArchiveURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com/app.tar.gz", true},
		{"https://example.com/app.tgz", true},
		{"https://example.com/app.tar.bz2", true},
		{"https://example.com/app.tar.xz", true},
		{"https://example.com/app.tar", true},
		{"https://example.com/app.gz", true},
		{"https://example.com/app.xz", true},
		{"https://example.com/app.tar.gz?token=abc", true},
		{"https://example.com/app.tar.gz#section", true},
		{"http://example.com/app.tar.gz", true},
		{"ftp://example.com/app.tar.gz", true},
		{"https://example.com/app.tar.zst", true},
		{"https://example.com/script.sh", false},
		{"https://example.com/config.json", false},
		{"https://example.com/page.html", false},
		{"/local/path/app.tar.gz", false},
		{"app.tar.gz", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := IsArchiveURL(tt.input); got != tt.want {
				t.Errorf("IsArchiveURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsTarExtract(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"short -x", []string{"-xf", "app.tar"}, true},
		{"combined -xzf", []string{"-xzf", "app.tar.gz"}, true},
		{"long --extract", []string{"--extract", "-f", "app.tar"}, true},
		{"long --get", []string{"--get", "-f", "app.tar"}, true},
		{"create -cf", []string{"-cf", "backup.tar", "/data"}, false},
		{"list -tf", []string{"-tf", "app.tar"}, false},
		{"no flags", []string{"app.tar"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := &CommandInfo{Name: "tar", Args: tt.args}
			if got := IsTarExtract(cmd); got != tt.want {
				t.Errorf("IsTarExtract(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestTarDestination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"short -C", []string{"-xf", "app.tar", "-C", "/opt"}, "/opt"},
		{"long --directory=", []string{"-xf", "app.tar", "--directory=/srv"}, "/srv"},
		{"long --directory space", []string{"-x", "--directory", "/var/lib"}, "/var/lib"},
		{"no destination", []string{"-xf", "app.tar"}, ""},
		{"-C at end (no value)", []string{"-xf", "app.tar", "-C"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := &CommandInfo{Name: "tar", Args: tt.args}
			if got := TarDestination(cmd); got != tt.want {
				t.Errorf("TarDestination(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestBasename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"/usr/src/app.tar", "app.tar"},
		{"foo/bar/app.tar", "app.tar"},
		{`build\foo\bar.tar.gz`, "bar.tar.gz"},
		{`"C:\Program Files\foo.tar.gz"`, "foo.tar.gz"},
		{"app.tar", "app.tar"},
		{`"/some/path"`, "path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := Basename(tt.input); got != tt.want {
				t.Errorf("Basename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDropQuotes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{`hello`, "hello"},
		{`""`, ""},
		{`"`, `"`},
		{``, ``},
		{`"mismatched'`, `"mismatched'`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := DropQuotes(tt.input); got != tt.want {
				t.Errorf("DropQuotes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
