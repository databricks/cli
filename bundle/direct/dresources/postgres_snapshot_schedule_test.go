package dresources

import "testing"

func TestSnapshotScheduleName(t *testing.T) {
	const branch = "projects/p/branches/b"
	want := branch + "/snapshot-schedule"

	cases := []string{
		branch,
		branch + "/",
		branch + "//",
	}
	for _, in := range cases {
		if got := snapshotScheduleName(in); got != want {
			t.Errorf("snapshotScheduleName(%q) = %q, want %q", in, got, want)
		}
	}
}
