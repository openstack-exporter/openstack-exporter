package exporters

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Feature names something a collector learns about its service when it binds
// to a cloud: an API extension, a backend kind, a negotiated version axis.
// Each collector declares its own constants and nothing outside the collector
// interprets them; the scheduler only compares them for equality.
type Feature string

// Version is an ordered major.minor pair, as OpenStack microversions are.
type Version struct {
	Major, Minor int
}

// ParseVersion parses "major.minor". It is the one place a version string is
// taken apart; everything downstream compares Version values.
func ParseVersion(s string) (Version, error) {
	major, minor, ok := strings.Cut(s, ".")
	if !ok {
		return Version{}, fmt.Errorf("version %q: want major.minor", s)
	}
	v := Version{}
	var err error
	if v.Major, err = strconv.Atoi(major); err != nil || v.Major < 0 {
		return Version{}, fmt.Errorf("version %q: bad major", s)
	}
	if v.Minor, err = strconv.Atoi(minor); err != nil || v.Minor < 0 {
		return Version{}, fmt.Errorf("version %q: bad minor", s)
	}
	return v, nil
}

func (v Version) String() string { return fmt.Sprintf("%d.%d", v.Major, v.Minor) }

// Compare orders versions: negative when v < o, zero when equal, positive
// when v > o.
func (v Version) Compare(o Version) int {
	if v.Major != o.Major {
		return v.Major - o.Major
	}
	return v.Minor - o.Minor
}

// Capabilities is the set of features a collector found at bind time. The
// zero value is empty and ready to use; a nil *Capabilities reads as empty.
type Capabilities struct {
	flags    map[Feature]struct{}
	versions map[Feature]Version
}

// Set records that feature f is present.
func (c *Capabilities) Set(f Feature) {
	if c.flags == nil {
		c.flags = make(map[Feature]struct{})
	}
	c.flags[f] = struct{}{}
}

// SetVersion records the value of a version axis, such as a service's
// negotiated microversion.
func (c *Capabilities) SetVersion(axis Feature, v Version) {
	if c.versions == nil {
		c.versions = make(map[Feature]Version)
	}
	c.versions[axis] = v
}

// Has reports whether feature f was set.
func (c *Capabilities) Has(f Feature) bool {
	if c == nil {
		return false
	}
	_, ok := c.flags[f]
	return ok
}

// Version returns the value of a version axis, if one was set.
func (c *Capabilities) Version(axis Feature) (Version, bool) {
	if c == nil {
		return Version{}, false
	}
	v, ok := c.versions[axis]
	return v, ok
}

// String lists the capabilities in a stable order, for bind-time logging.
func (c *Capabilities) String() string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, len(c.flags)+len(c.versions))
	for f := range c.flags {
		parts = append(parts, string(f))
	}
	for axis, v := range c.versions {
		parts = append(parts, string(axis)+"="+v.String())
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// Requirement is a condition on Capabilities that a graph node declares. A
// node runs only when every one of its requirements holds; XOR group members
// are chosen by how many requirements they declare, see Graph.
//
// Implementations are small comparable values so that a schedule can report
// what it selected and why.
type Requirement interface {
	Satisfied(c *Capabilities) bool
	String() string
}

// Has requires a feature to be present.
type Has Feature

func (r Has) Satisfied(c *Capabilities) bool { return c.Has(Feature(r)) }
func (r Has) String() string                 { return "has " + string(r) }

// AtLeast requires a version axis to be set and >= Min.
type AtLeast struct {
	Axis Feature
	Min  Version
}

func (r AtLeast) Satisfied(c *Capabilities) bool {
	v, ok := c.Version(r.Axis)
	return ok && v.Compare(r.Min) >= 0
}

func (r AtLeast) String() string { return string(r.Axis) + ">=" + r.Min.String() }

// Below requires a version axis to be set and < Max. Together with AtLeast it
// makes the ranges of XOR group members disjoint.
type Below struct {
	Axis Feature
	Max  Version
}

func (r Below) Satisfied(c *Capabilities) bool {
	v, ok := c.Version(r.Axis)
	return ok && v.Compare(r.Max) < 0
}

func (r Below) String() string { return string(r.Axis) + "<" + r.Max.String() }

func satisfied(reqs []Requirement, c *Capabilities) bool {
	for _, r := range reqs {
		if !r.Satisfied(c) {
			return false
		}
	}
	return true
}

func requirementsString(reqs []Requirement) string {
	if len(reqs) == 0 {
		return "default"
	}
	parts := make([]string, len(reqs))
	for i, r := range reqs {
		parts[i] = r.String()
	}
	return strings.Join(parts, " && ")
}
