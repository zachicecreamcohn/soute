package paths

import "testing"

func TestCleanInput(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"C:\show.qlab5"`, `C:\show.qlab5`},
		{`'C:\show.qlab5'`, `C:\show.qlab5`},
		{`  C:\show.qlab5  `, `C:\show.qlab5`},
		{`C:\show.qlab5`, `C:\show.qlab5`},
		{`C:\foo"bar`, `C:\foo"bar`}, // inner quote is left alone
		{``, ``},
	}
	for _, c := range cases {
		if got := CleanInput(c.in); got != c.want {
			t.Errorf("CleanInput(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
