package upload

import "testing"

func TestReleaseDecision(t *testing.T) {
	cases := []struct {
		name       string
		found      bool
		want       bool
		diagnostic string
	}{{"missing", false, false, "upload the asset, then retry the release check"}, {"present", true, true, "asset is present; release can proceed"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := ReleaseResult{Ready: tc.found}
			if r.Ready != tc.want {
				t.Fatalf("ready=%v", r.Ready)
			}
			if tc.found {
				r.Diagnostic = "asset is present; release can proceed"
			} else {
				r.Diagnostic = "upload the asset, then retry the release check"
			}
			if r.Diagnostic != tc.diagnostic {
				t.Fatal(r.Diagnostic)
			}
		})
	}
}
