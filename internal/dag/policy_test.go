// Copyright © 2019 VMware
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

	ingressroutev1 "github.com/projectcontour/contour/apis/contour/v1beta1"
	projcontour "github.com/projectcontour/contour/apis/projectcontour/v1"
	"github.com/projectcontour/contour/internal/assert"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRetryPolicyIngress(t *testing.T) {
	tests := map[string]struct {
		i       *v1beta1.Ingress
		options RouteOptions
		limits  RouteLimits
		want    *RetryPolicy
	}{
		"no anotations": {
			i:    &v1beta1.Ingress{},
			want: nil,
		},
		"retry-on": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on": "5xx",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn: "5xx",
			},
		},
		"retry-on-defaulted": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on": "5xx",
					},
				},
			},
			options: RouteOptions{
				NumRetries:    3,
				PerTryTimeout: 10 * time.Second,
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    3,
				PerTryTimeout: 10 * time.Second,
			},
		},
		"retry-on-limited": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":        "5xx",
						"projectcontour.io/num-retries":     "99",
						"projectcontour.io/per-try-timeout": "10m",
					},
				},
			},
			limits: RouteLimits{
				NumRetries:    3,
				PerTryTimeout: 10 * time.Second,
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    3,
				PerTryTimeout: 10 * time.Second,
			},
		},
		"explicitly zero retries": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":    "5xx",
						"projectcontour.io/num-retries": "0",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 0,
			},
		},
		"legacy explicitly zero retries": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":     "5xx",
						"contour.heptio.com/num-retries": "0",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 0,
			},
		},
		"num-retries": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":    "5xx",
						"projectcontour.io/num-retries": "7",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 7,
			},
		},
		"legacy num-retries": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":     "5xx",
						"contour.heptio.com/num-retries": "7",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 7,
			},
		},
		"no retry count, per try timeout": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":        "5xx",
						"projectcontour.io/per-try-timeout": "10s",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    0,
				PerTryTimeout: 10 * time.Second,
			},
		},
		"no retry count, legacy per try timeout": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":         "5xx",
						"contour.heptio.com/per-try-timeout": "10s",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    0,
				PerTryTimeout: 10 * time.Second,
			},
		},
		"explicit 0s timeout": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":        "5xx",
						"projectcontour.io/per-try-timeout": "0s",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    0,
				PerTryTimeout: 0 * time.Second,
			},
		},
		"legacy explicit 0s timeout": {
			i: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"projectcontour.io/retry-on":         "5xx",
						"contour.heptio.com/per-try-timeout": "0s",
					},
				},
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    0,
				PerTryTimeout: 0 * time.Second,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ingressRetryPolicy(tc.i, tc.options, tc.limits)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRetryPolicyIngressRoute(t *testing.T) {
	tests := map[string]struct {
		rp   *projcontour.RetryPolicy
		want *RetryPolicy
	}{
		"nil retry policy": {
			rp:   nil,
			want: nil,
		},
		"empty policy": {
			rp: &projcontour.RetryPolicy{},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 1,
			},
		},
		"explicitly zero retries": {
			rp: &projcontour.RetryPolicy{
				NumRetries: 0, // zero value for NumRetries
			},
			want: &RetryPolicy{
				RetryOn:    "5xx",
				NumRetries: 1,
			},
		},
		"no retry count, per try timeout": {
			rp: &projcontour.RetryPolicy{
				PerTryTimeout: "10s",
			},
			want: &RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    1,
				PerTryTimeout: 10 * time.Second,
			},
		},
		"explicit 0s timeout": {
			rp: &projcontour.RetryPolicy{
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
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTimeoutPolicyIngressRoute(t *testing.T) {
	tests := map[string]struct {
		tp      *ingressroutev1.TimeoutPolicy
		options RouteOptions
		limits  RouteLimits
		want    *TimeoutPolicy
	}{
		"nil timeout policy": {
			tp:   nil,
			want: nil,
		},
		"empty timeout policy": {
			tp: &ingressroutev1.TimeoutPolicy{},
			want: &TimeoutPolicy{
				ResponseTimeout: 0 * time.Second,
			},
		},
		"valid request timeout": {
			tp: &ingressroutev1.TimeoutPolicy{
				Request: "1m30s",
			},
			want: &TimeoutPolicy{
				ResponseTimeout: 90 * time.Second,
			},
		},
		"invalid request timeout": {
			tp: &ingressroutev1.TimeoutPolicy{
				Request: "90", // 90 what?
			},
			want: &TimeoutPolicy{
				// the documentation for an invalid timeout says the duration will
				// be undefined. In practice we take the spec from the
				// contour.heptio.com/request-timeout annotation, which is defined
				// to choose infinite when its valid cannot be parsed.
				ResponseTimeout: -1,
			},
		},
		"infinite request timeout": {
			tp: &ingressroutev1.TimeoutPolicy{
				Request: "infinite",
			},
			want: &TimeoutPolicy{
				ResponseTimeout: -1,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ingressrouteTimeoutPolicy(tc.tp, tc.options, tc.limits)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTimeoutPolicy(t *testing.T) {
	tests := map[string]struct {
		tp      *projcontour.TimeoutPolicy
		options RouteOptions
		limits  RouteLimits
		want    *TimeoutPolicy
	}{
		"nil timeout policy": {
			tp:   nil,
			want: nil,
		},
		"empty timeout policy": {
			tp: &projcontour.TimeoutPolicy{},
			want: &TimeoutPolicy{
				ResponseTimeout: 0 * time.Second,
			},
		},
		"defaulted response timeout": {
			tp: &projcontour.TimeoutPolicy{},
			options: RouteOptions{
				ResponseTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				ResponseTimeout: 120 * time.Second,
			},
		},
		"defaulted invalid response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Response: "90", // 90 what?
			},
			options: RouteOptions{
				ResponseTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				ResponseTimeout: 120 * time.Second,
			},
		},
		"valid response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Response: "1m30s",
			},
			want: &TimeoutPolicy{
				ResponseTimeout: 90 * time.Second,
			},
		},
		"limit response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Response: "1m30s",
			},
			limits: RouteLimits{
				ResponseTimeout: 30 * time.Second,
			},
			want: &TimeoutPolicy{
				ResponseTimeout: 30 * time.Second,
			},
		},
		"invalid response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Response: "90", // 90 what?
			},
			want: &TimeoutPolicy{
				// the documentation for an invalid timeout says the duration will
				// be undefined. In practice we take the spec from the
				// contour.heptio.com/request-timeout annotation, which is defined
				// to choose infinite when its valid cannot be parsed.
				ResponseTimeout: -1,
			},
		},
		"infinite response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Response: "infinite",
			},
			want: &TimeoutPolicy{
				ResponseTimeout: -1,
			},
		},
		"limit infinite response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Response: "infinite",
			},
			limits: RouteLimits{
				ResponseTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				ResponseTimeout: 120 * time.Second,
			},
		},
		"defaulted idle timeout": {
			tp: &projcontour.TimeoutPolicy{},
			options: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 120 * time.Second,
			},
		},
		"defaulted invalid idle timeout": {
			tp: &projcontour.TimeoutPolicy{
				Idle: "900", // 900 what?
			},
			options: RouteOptions{
				IdleTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 120 * time.Second,
			},
		},
		"valid idle timeout": {
			tp: &projcontour.TimeoutPolicy{
				Idle: "900s",
			},
			want: &TimeoutPolicy{
				IdleTimeout: 900 * time.Second,
			},
		},
		"limit idle timeout": {
			tp: &projcontour.TimeoutPolicy{
				Idle: "900s",
			},
			limits: RouteLimits{
				IdleTimeout: 300 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 300 * time.Second,
			},
		},
		"invalid idle timeout": {
			tp: &projcontour.TimeoutPolicy{
				Idle: "900", // 90 what?
			},
			want: &TimeoutPolicy{
				// the documentation for an invalid timeout says the duration will
				// be undefined. In practice we take the spec from the
				// contour.heptio.com/request-timeout annotation, which is defined
				// to choose infinite when its valid cannot be parsed.
				IdleTimeout: -1,
			},
		},
		"infinite idle timeout": {
			tp: &projcontour.TimeoutPolicy{
				Idle: "infinite",
			},
			want: &TimeoutPolicy{
				IdleTimeout: -1,
			},
		},
		"limit idle response timeout": {
			tp: &projcontour.TimeoutPolicy{
				Idle: "infinite",
			},
			limits: RouteLimits{
				IdleTimeout: 300 * time.Second,
			},
			want: &TimeoutPolicy{
				IdleTimeout: 300 * time.Second,
			},
		},
		"defaulted max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{},
			options: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 120 * time.Second,
			},
		},
		"defaulted invalid max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{
				MaxGrpc: "90", // 90 what?
			},
			options: RouteOptions{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 120 * time.Second,
			},
		},
		"max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{
				MaxGrpc: "900s",
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 900 * time.Second,
			},
		},
		"limit max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{
				MaxGrpc: "900s",
			},
			limits: RouteLimits{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 120 * time.Second,
			},
		},
		"invalid max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{
				MaxGrpc: "90", // 90 what?
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: -1,
			},
		},
		"infinite max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{
				MaxGrpc: "infinite",
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: -1,
			},
		},
		"limit infinite max grpc timeout": {
			tp: &projcontour.TimeoutPolicy{
				MaxGrpc: "infinite",
			},
			limits: RouteLimits{
				MaxGrpcTimeout: 120 * time.Second,
			},
			want: &TimeoutPolicy{
				MaxGrpcTimeout: 120 * time.Second,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := timeoutPolicy(tc.tp, tc.options, tc.limits)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoadBalancerPolicy(t *testing.T) {
	tests := map[string]struct {
		lbp  *projcontour.LoadBalancerPolicy
		want string
	}{
		"nil": {
			lbp:  nil,
			want: "",
		},
		"empty": {
			lbp:  &projcontour.LoadBalancerPolicy{},
			want: "",
		},
		"WeightedLeastRequest": {
			lbp: &projcontour.LoadBalancerPolicy{
				Strategy: "WeightedLeastRequest",
			},
			want: "WeightedLeastRequest",
		},
		"Random": {
			lbp: &projcontour.LoadBalancerPolicy{
				Strategy: "Random",
			},
			want: "Random",
		},
		"Cookie": {
			lbp: &projcontour.LoadBalancerPolicy{
				Strategy: "Cookie",
			},
			want: "Cookie",
		},
		"unknown": {
			lbp: &projcontour.LoadBalancerPolicy{
				Strategy: "please",
			},
			want: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := loadBalancerPolicy(tc.lbp)
			assert.Equal(t, tc.want, got)
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
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseTimeoutWithDefault(t *testing.T) {
	for name, tc := range map[string]struct {
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
	} {
		t.Run(name, func(t *testing.T) {
			got := parseTimeoutWithDefault(tc.duration, tc.alt)
			assert.Equal(t, tc.want, got)
		})
	}
}
