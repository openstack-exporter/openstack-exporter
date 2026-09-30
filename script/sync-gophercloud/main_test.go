package main

import (
	"strings"
	"testing"
)

func TestExtract(t *testing.T) {
	source := []byte(`package upstream
import (
 "fmt"
 "net/http"
 "github.com/gophercloud/gophercloud/v2"
 th "github.com/gophercloud/gophercloud/v2/testhelper"
 "github.com/gophercloud/gophercloud/v2/internal/acceptance/tools"
)
// Choice retains its fields and comments.
type Choice struct {
 // Name is required.
 Name string
}
// Keep retains its body comment.
func Keep() {
 // Preserve me.
 th.AssertEquals(nil, "x", tools.RandomString("", 1))
 fmt.Println(Choice{Name: "x"}, gophercloud.Enabled)
}
func Drop() { http.Get("unused") }
`)
	got, err := extract(source, "local", []string{"Choice", "Keep"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"github.com/gophercloud/gophercloud/v2"`, "package local", "// Name is required.", "Name string",
		"// Keep retains its body comment.", "// Preserve me.",
		`th "github.com/gophercloud/gophercloud/v2/testhelper"`,
		`"github.com/openstack-exporter/openstack-exporter/integration/tools"`,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"func Drop", "net/http", "internal/acceptance"} {
		if strings.Contains(string(got), unwanted) {
			t.Errorf("output contains %q:\n%s", unwanted, got)
		}
	}
	if _, err := extract(source, "local", []string{"Missing"}); err == nil {
		t.Fatal("missing declarations must fail rather than silently disappear")
	}
}
