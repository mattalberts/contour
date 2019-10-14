package main

import (
	"fmt"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type resource struct {
	name, namespace string
}

func (r *resource) Set(val string) error {
	if len(val) > 0 {
		parts := strings.SplitN(strings.TrimSpace(val), "/", 3)
		if len(parts) != 2 {
			return fmt.Errorf("expected '<namespace>/<name>' got '%s'", val)
		} else if parts[0] == "" {
			return fmt.Errorf("expected '<namespace>/<name>' got '%s'", val)
		} else if parts[1] == "" {
			return fmt.Errorf("expected '<namespace>/<name>' got '%s'", val)
		}
		r.namespace, r.name = parts[0], parts[1]
	}
	return nil
}

func (r *resource) String() (val string) {
	if r.namespace != "" && r.name != "" {
		val = r.namespace + "/" + r.name
	}
	return
}

func metaMixin(s kingpin.Settings) (val *resource) {
	val = &resource{}
	s.SetValue(val)
	return
}
