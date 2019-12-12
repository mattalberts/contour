package main

import (
	"fmt"
	"reflect"
	"testing"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func TestMetaMixin(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		in  string
		out *resource
		err error
	}{
		"resource-empty": {
			in:  "",
			out: &resource{},
			err: nil,
		},
		"resource-no-name": {
			in:  "a/",
			out: &resource{},
			err: fmt.Errorf("expected '<namespace>/<name>' got '%s'", "a/"),
		},
		"resource-no-namespace": {
			in:  "/b",
			out: &resource{},
			err: fmt.Errorf("expected '<namespace>/<name>' got '%s'", "/b"),
		},
		"resource-too-many-parts": {
			in:  "a/b/c",
			out: &resource{},
			err: fmt.Errorf("expected '<namespace>/<name>' got '%s'", "a/b/c"),
		},
		"resource-okay": {
			in: "a/b",
			out: &resource{
				Namespace: "a",
				Name:      "b",
			},
			err: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			app := kingpin.New(name, name)
			out := metaMixin(app.Flag("o", "test"))
			_, err := app.Parse([]string{"--o=" + tc.in})

			if !reflect.DeepEqual(out, tc.out) {
				t.Fatalf("expected: %q, got: %q", tc.out, out)
			}
			if !reflect.DeepEqual(err, tc.err) {
				t.Fatalf("expected: %q, got: %q", tc.err, err)
			}
		})
	}
}

func TestMetaString(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		in  *resource
		out string
	}{
		"resource-empty": {
			in:  &resource{},
			out: "",
		},
		"resource-no-name": {
			in: &resource{
				Namespace: "a",
			},
			out: "",
		},
		"resource-no-namespace": {
			in: &resource{
				Name: "b",
			},
			out: "",
		},
		"resource-okay": {
			in: &resource{
				Namespace: "a",
				Name:      "b",
			},
			out: "a/b",
		},
	} {
		t.Run(name, func(t *testing.T) {
			out := tc.in.String()
			if !reflect.DeepEqual(out, tc.out) {
				t.Fatalf("expected: %q, got: %q", tc.out, out)
			}
		})
	}
}
