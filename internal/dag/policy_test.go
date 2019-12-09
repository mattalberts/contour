// Copyright © 2019 Heptio
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

package dag

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/heptio/contour/apis/contour/v1beta1"
)

func TestRetryPolicyIngressRoute(t *testing.T) {
	tests := map[string]struct {
		rp   *v1beta1.RetryPolicy
		want *RetryPolicy
	}{
		"nil retry policy": {
			rp:   nil,
			want: nil,
		},
		"empty policy": {
			rp: &v1beta1.RetryPolicy{},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 1,
			},
		},
		"explicitly zero retries": {
			rp: &v1beta1.RetryPolicy{
				NumRetries: 0, // zero value for NumRetries
			},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 1,
			},
		},
		"no retry count, per try timeout": {
			rp: &v1beta1.RetryPolicy{
				PerTryTimeout: "10s",
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    1,
				PerTryTimeout: 10 * time.Second,
			},
		},
		"explicit 0s timeout": {
			rp: &v1beta1.RetryPolicy{
				PerTryTimeout: "0s",
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    1,
				PerTryTimeout: 0 * time.Second,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := retryPolicy(tc.rp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTimeoutPolicyIngressRoute(t *testing.T) {
	tests := map[string]struct {
		tp     *v1beta1.TimeoutPolicy
		opts   RouteOptions
		limits RouteLimits
		want   *TimeoutPolicy
	}{
		"nil timeout policy": {
			tp:   nil,
			want: nil,
		},
		"empty timeout policy": {
			tp: &v1beta1.TimeoutPolicy{},
			want: &TimeoutPolicy{
				Timeout: 0 * time.Second,
			},
		},
		"limit request timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Request: "1m30s",
			},
			limits: RouteLimits{
				RequestTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				Timeout: 30 * time.Second,
			},
		},
		"valid request timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Request: "1m30s",
			},
			want: &TimeoutPolicy{
				Timeout: 90 * time.Second,
			},
		},
		"invalid request timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Request: "90", // 90 what?
			},
			want: &TimeoutPolicy{
				// the documentation for an invalid timeout says the duration will
				// be undefined. In practice we take the spec from the
				// contour.heptio.com/request-timeout annotation, which is defined
				// to choose infinite when its valid cannot be parsed.
				Timeout: -1,
			},
		},
		"infinite request timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Request: "infinity",
			},
			want: &TimeoutPolicy{
				Timeout: -1,
			},
		},
		"limit infinite request timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Request: "infinity",
			},
			limits: RouteLimits{
				RequestTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				Timeout: 30 * time.Second,
			},
		},
		"default idle timeout": {
			tp: &v1beta1.TimeoutPolicy{},
			opts: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 120 * time.Second,
			},
		},
		"limit idle timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Idle: "1m30s",
			},
			limits: RouteLimits{
				IdleTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 30 * time.Second,
			},
		},
		"valid idle timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Idle: "1m30s",
			},
			opts: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 90 * time.Second,
			},
		},
		"invalid idle timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Idle: "90",
			},
			opts: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 120 * time.Second,
			},
		},
		"infinite idle timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Idle: "infinity",
			},
			opts: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: -1,
			},
		},
		"limit infinite idle timeout": {
			tp: &v1beta1.TimeoutPolicy{
				Idle: "infinity",
			},
			opts: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			limits: RouteLimits{
				IdleTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 30 * time.Second,
			},
		},
		"default max_grpc timeout": {
			tp: &v1beta1.TimeoutPolicy{},
			opts: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 120 * time.Second,
			},
		},
		"limit max_grpc timeout": {
			tp: &v1beta1.TimeoutPolicy{
				MaxGrpc: "1m30s",
			},
			limits: RouteLimits{
				MaxGrpcTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 30 * time.Second,
			},
		},
		"valid max_grpc timeout": {
			tp: &v1beta1.TimeoutPolicy{
				MaxGrpc: "1m30s",
			},
			opts: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 90 * time.Second,
			},
		},
		"invalid max_grpc timeout": {
			tp: &v1beta1.TimeoutPolicy{
				MaxGrpc: "90",
			},
			opts: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 120 * time.Second,
			},
		},
		"infinite max_grpc timeout": {
			tp: &v1beta1.TimeoutPolicy{
				MaxGrpc: "infinity",
			},
			opts: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: -1,
			},
		},
		"limit infinite max_grpc timeout": {
			tp: &v1beta1.TimeoutPolicy{
				MaxGrpc: "infinity",
			},
			opts: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			limits: RouteLimits{
				MaxGrpcTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 30 * time.Second,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := timeoutPolicy(tc.tp, tc.opts, tc.limits)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestParseTimeout(t *testing.T) {
	tests := map[string]struct {
		duration string
		want     time.Duration
	}{
		"empty": {
			duration: "",
			want:     0,
		},
		"infinity": {
			duration: "infinity",
			want:     -1,
		},
		"10 seconds": {
			duration: "10s",
			want:     10 * time.Second,
		},
		"invalid": {
			duration: "10", // 10 what?
			want:     -1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseTimeout(tc.duration)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestParseTimeoutWithDefault(t *testing.T) {
	tests := map[string]struct {
		duration string
		alt      time.Duration
		want     time.Duration
	}{
		"empty": {
			duration: "",
			want:     0,
		},
		"empty with non-zero default": {
			duration: "",
			alt:      5 * time.Second,
			want:     5 * time.Second,
		},
		"infinity": {
			duration: "infinity",
			want:     -1,
		},
		"10 seconds": {
			duration: "10s",
			want:     10 * time.Second,
		},
		"invalid": {
			duration: "10", // 10 what?
			want:     -1,
		},
		"invalid with non-zero default": {
			duration: "10", // 10 what?
			alt:      5 * time.Second,
			want:     5 * time.Second,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseTimeoutWithDefault(tc.duration, tc.alt)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
