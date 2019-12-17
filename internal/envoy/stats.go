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

package envoy

import (
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	http "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/projectcontour/contour/internal/protobuf"
)

// StatsOptions defines options for the stats listener
type StatsOptions struct {
	ListenerOptions
	HTTPConnectionOptions
}

// StatsListener returns a *v2.Listener configured to serve prometheus
// metrics on /stats.
func StatsListener(address string, port int, options StatsOptions) *v2.Listener {
	return &v2.Listener{
		Name:                             "stats-health",
		Address:                          SocketAddress(address, port),
		PerConnectionBufferLimitBytes:    uint32ptoto(options.PerConnectionBufferLimitBytes),
		ListenerFiltersTimeout:           durationproto(options.ListenerFiltersTimeout),
		ContinueOnListenerFiltersTimeout: options.ContinueOnListenerFiltersTimeout,
		TcpFastOpenQueueLength:           uint32ptoto(options.TCPFastOpenQueueLength),
		FilterChains: FilterChains(
			&envoy_api_v2_listener.Filter{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &envoy_api_v2_listener.Filter_TypedConfig{
					TypedConfig: toAny(&http.HttpConnectionManager{
						StatPrefix: "stats",
						RouteSpecifier: &http.HttpConnectionManager_RouteConfig{
							RouteConfig: &v2.RouteConfiguration{
								VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
									Name:    "backend",
									Domains: []string{"*"},
									Routes: []*envoy_api_v2_route.Route{{
										Match: &envoy_api_v2_route.RouteMatch{
											PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{
												Prefix: "/ready",
											},
										},
										Action: &envoy_api_v2_route.Route_Route{
											Route: &envoy_api_v2_route.RouteAction{
												ClusterSpecifier: &envoy_api_v2_route.RouteAction_Cluster{
													Cluster: "service-stats",
												},
											},
										},
									}, {
										Match: &envoy_api_v2_route.RouteMatch{
											PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{
												Prefix: "/stats",
											},
										},
										Action: &envoy_api_v2_route.Route_Route{
											Route: &envoy_api_v2_route.RouteAction{
												ClusterSpecifier: &envoy_api_v2_route.RouteAction_Cluster{
													Cluster: "service-stats",
												},
											},
										},
									},
									},
								}},
							},
						},
						HttpFilters: []*http.HttpFilter{{
							Name: wellknown.Router,
						}},
						CommonHttpProtocolOptions: &envoy_api_v2_core.HttpProtocolOptions{
							IdleTimeout: ptypes.DurationProto(tvd(options.IdleTimeout, 60*time.Second)),
						},
						NormalizePath:       protobuf.Bool(true),
						DelayedCloseTimeout: durationproto(options.DelayedCloseTimeout),
						DrainTimeout:        durationproto(options.DrainTimeout),
						RequestTimeout:      durationproto(options.RequestTimeout),
						StreamIdleTimeout:   durationproto(options.StreamIdleTimeout),
					}),
				},
			},
		),
	}
}
