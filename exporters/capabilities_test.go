package exporters

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in      string
		want    Version
		wantErr bool
	}{
		{in: "2.47", want: Version{2, 47}},
		{in: "1.0", want: Version{1, 0}},
		{in: "2", wantErr: true},
		{in: "2.x", wantErr: true},
		{in: "-1.2", wantErr: true},
		{in: "2.-3", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseVersion(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b Version
		sign int
	}{
		{Version{2, 47}, Version{2, 47}, 0},
		{Version{2, 47}, Version{2, 33}, 1},
		{Version{2, 33}, Version{2, 47}, -1},
		{Version{3, 0}, Version{2, 99}, 1},
		{Version{1, 99}, Version{2, 0}, -1},
	}
	for _, tt := range tests {
		got := tt.a.Compare(tt.b)
		switch {
		case tt.sign == 0 && got != 0, tt.sign > 0 && got <= 0, tt.sign < 0 && got >= 0:
			t.Errorf("%v.Compare(%v) = %d, want sign %d", tt.a, tt.b, got, tt.sign)
		}
	}
	if got := (Version{2, 47}).String(); got != "2.47" {
		t.Errorf("String() = %q, want 2.47", got)
	}
}

func TestRequirementsAgainstEmptyCapabilities(t *testing.T) {
	const axis Feature = "svc.microversion"
	reqs := []Requirement{Has("svc.ext"), AtLeast{axis, Version{2, 1}}, Below{axis, Version{2, 1}}}

	for _, caps := range []*Capabilities{nil, {}} {
		for _, r := range reqs {
			if r.Satisfied(caps) {
				t.Errorf("%s satisfied by empty capabilities %v", r, caps)
			}
		}
		if !satisfied(nil, caps) {
			t.Errorf("empty requirement list not satisfied by %v", caps)
		}
	}
}

func TestRequirements(t *testing.T) {
	const (
		ext  Feature = "svc.ext"
		axis Feature = "svc.microversion"
	)
	var caps Capabilities
	caps.Set(ext)
	caps.SetVersion(axis, Version{2, 47})

	tests := []struct {
		req  Requirement
		want bool
		str  string
	}{
		{Has(ext), true, "has svc.ext"},
		{Has("svc.other"), false, "has svc.other"},
		{AtLeast{axis, Version{2, 47}}, true, "svc.microversion>=2.47"},
		{AtLeast{axis, Version{2, 48}}, false, "svc.microversion>=2.48"},
		{AtLeast{"svc.unset", Version{0, 0}}, false, "svc.unset>=0.0"},
		{Below{axis, Version{2, 48}}, true, "svc.microversion<2.48"},
		{Below{axis, Version{2, 47}}, false, "svc.microversion<2.47"},
	}
	for _, tt := range tests {
		if got := tt.req.Satisfied(&caps); got != tt.want {
			t.Errorf("%s Satisfied = %v, want %v", tt.req, got, tt.want)
		}
		if got := tt.req.String(); got != tt.str {
			t.Errorf("String() = %q, want %q", got, tt.str)
		}
	}

	if got, want := caps.String(), "svc.ext, svc.microversion=2.47"; got != want {
		t.Errorf("Capabilities.String() = %q, want %q", got, want)
	}
	if got := requirementsString(nil); got != "default" {
		t.Errorf("requirementsString(nil) = %q, want default", got)
	}
	if got, want := requirementsString([]Requirement{Has(ext), Below{axis, Version{3, 0}}}), "has svc.ext && svc.microversion<3.0"; got != want {
		t.Errorf("requirementsString = %q, want %q", got, want)
	}
}
