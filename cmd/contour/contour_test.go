// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseRootNamespaces(t *testing.T) {
	tests := map[string]struct {
		input string
		want  []string
	}{
		"empty": {
			input: "",
			want:  nil,
		},
		"one value": {
			input: "heptio-contour",
			want:  []string{"heptio-contour"},
		},
		"multiple, easy": {
			input: "prod1,prod2,prod3",
			want:  []string{"prod1", "prod2", "prod3"},
		},
		"multiple, hard": {
			input: "prod1, prod2, prod3 ",
			want:  []string{"prod1", "prod2", "prod3"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseRootNamespaces(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expected: %q, got: %q", tc.want, got)
			}
		})
	}
}

func TestLogrusLevel(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		in  int
		out logrus.Level
	}{
		"verbose-flag-lt-0": {
			in:  -100,
			out: logrus.PanicLevel,
		},
		"verbose-flag-0": {
			in:  0,
			out: logrus.PanicLevel,
		},
		"verbose-flag-1": {
			in:  1,
			out: logrus.FatalLevel,
		},
		"verbose-flag-2": {
			in:  2,
			out: logrus.ErrorLevel,
		},
		"verbose-flag-3": {
			in:  3,
			out: logrus.WarnLevel,
		},
		"verbose-flag-4": {
			in:  4,
			out: logrus.InfoLevel,
		},
		"verbose-flag-5": {
			in:  5,
			out: logrus.DebugLevel,
		},
		"verbose-flag-100": {
			in:  100,
			out: logrus.DebugLevel,
		},
	} {
		t.Run(name, func(t *testing.T) {
			out := logruslevel(tc.in)
			if !reflect.DeepEqual(out, tc.out) {
				t.Fatalf("expected: %q, got: %q", tc.out, out)
			}
		})
	}
}
