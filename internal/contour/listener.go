// Copyright © 2017 Heptio
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

package contour

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/gogo/protobuf/types"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

const (
	ENVOY_HTTP_LISTENER            = "ingress_http"
	ENVOY_HTTPS_LISTENER           = "ingress_https"
	DEFAULT_HTTP_LISTENER_ADDRESS  = "0.0.0.0"
	DEFAULT_HTTP_LISTENER_PORT     = 8080
	DEFAULT_HTTPS_LISTENER_ADDRESS = DEFAULT_HTTP_LISTENER_ADDRESS
	DEFAULT_HTTPS_LISTENER_PORT    = 8443

	router     = "envoy.router"
	httpFilter = "envoy.http_connection_manager"
	accessLog  = "envoy.file_access_log"
)

// ListenerCache manages the contents of the gRPC LDS cache.
type ListenerCache struct {
	// Envoy's HTTP (non TLS) listener address.
	// If not set, defaults to DEFAULT_HTTP_LISTENER_ADDRESS.
	HTTPAddress string

	// Envoy's HTTP (non TLS) listener port.
	// If not set, defaults to DEFAULT_HTTP_LISTENER_PORT.
	HTTPPort int

	// Envoy's HTTPS (TLS) listener address.
	// If not set, defaults to DEFAULT_HTTPS_LISTENER_ADDRESS.
	HTTPSAddress string

	// Envoy's HTTPS (TLS) listener port.
	// If not set, defaults to DEFAULT_HTTPS_LISTENER_PORT.
	HTTPSPort int

	// UseProxyProto configurs all listeners to expect a PROXY protocol
	// V1 header on new connections.
	// If not set, defaults to false.
	UseProxyProto bool

	// EnableTracing configures all listeners to emit tracing spans on
	// egress (request sent to upstream).
	// If not set, defaults to false.
	EnableTracing bool

	// DefaultTLSSecretName defines a default certificate to use for tls
	// termination. An Ingress spec secretName overrides this default.
	// If not set, defaults to ""
	DefaultTLSSecretName string

	// DefaultTLSSecretNamespace defines a default certificate to use for
	// tls termination. An Ingress spec secretName overrides this default.
	// If not set, defaults to ""
	DefaultTLSSecretNamespace string

	listenerCache
	Cond
}

// recomputeListeners recomputes the ingress_http and ingress_https listeners
// and notifies the watchers any change.
func (lc *ListenerCache) recomputeListeners(ingresses map[metadata]*v1beta1.Ingress, secrets map[metadata]*v1.Secret) {
	add, remove := lc.recomputeListener0(ingresses)                   // recompute ingress_http
	ssladd, sslremove := lc.recomputeTLSListener0(ingresses, secrets) // recompute ingress_https

	add = append(add, ssladd...)
	remove = append(remove, sslremove...)
	lc.Add(add...)
	lc.Remove(remove...)

	if len(add) > 0 || len(remove) > 0 {
		lc.Notify()
	}
}

// recomputeTLSListener recomputes the ingress_https listener and notifies the watchers
// of any change.
func (lc *ListenerCache) recomputeTLSListener(ingresses map[metadata]*v1beta1.Ingress, secrets map[metadata]*v1.Secret) {
	ssladd, sslremove := lc.recomputeTLSListener0(ingresses, secrets) // recompute ingress_https
	lc.Add(ssladd...)
	lc.Remove(sslremove...)
	if len(ssladd) > 0 || len(sslremove) > 0 {
		lc.Notify()
	}
}

// recomputeListener recomputes the non SSL listener for port 8080 using the list of ingresses provided.
// recomputeListener returns a slice of listeners to be added to the cache, and a slice of names of listeners
// to be removed.
func (lc *ListenerCache) recomputeListener0(ingresses map[metadata]*v1beta1.Ingress) ([]*v2.Listener, []string) {
	l := &v2.Listener{
		Name:    ENVOY_HTTP_LISTENER,
		Address: socketaddress(lc.httpAddress(), lc.httpPort()),
	}

	var valid int
	for _, i := range ingresses {
		if httpAllowed(i) {
			valid++
		}
	}
	if valid > 0 {
		l.FilterChains = []listener.FilterChain{
			filterchain(lc.UseProxyProto, httpfilter(ENVOY_HTTP_LISTENER, lc.EnableTracing)),
		}
	}
	// TODO(dfc) some annotations may require the Ingress to no appear on
	// port 80, therefore may result in an empty effective set of ingresses.
	switch len(l.FilterChains) {
	case 0:
		// no ingresses registered, remove this listener.
		return nil, []string{l.Name}
	default:
		// at least one ingress registered, refresh listener
		return []*v2.Listener{l}, nil
	}
}

// httpAddress returns the port for the HTTP (non TLS)
// listener or DEFAULT_HTTP_LISTENER_ADDRESS if not configured.
func (lc *ListenerCache) httpAddress() string {
	if lc.HTTPAddress != "" {
		return lc.HTTPAddress
	}
	return DEFAULT_HTTP_LISTENER_ADDRESS
}

// httpPort returns the port for the HTTP (non TLS)
// listener or DEFAULT_HTTP_LISTENER_PORT if not configured.
func (lc *ListenerCache) httpPort() uint32 {
	if lc.HTTPPort != 0 {
		return uint32(lc.HTTPPort)
	}
	return DEFAULT_HTTP_LISTENER_PORT
}

// recomputeTLSListener0 recomputes the SSL listener for port 8443
// using the list of ingresses and secrets provided.
// recomputeListener returns a slice of listeners to be added to the cache,
// and a slice of names of listeners to be removed. If the list of
// TLS enabled listeners is zero, the listener is removed.
func (lc *ListenerCache) recomputeTLSListener0(ingresses map[metadata]*v1beta1.Ingress, secrets map[metadata]*v1.Secret) ([]*v2.Listener, []string) {
	l := &v2.Listener{
		Name:    ENVOY_HTTPS_LISTENER,
		Address: socketaddress(lc.httpsAddress(), lc.httpsPort()),
	}

	filters := []listener.Filter{
		httpfilter(ENVOY_HTTPS_LISTENER, lc.EnableTracing),
	}

	for _, i := range ingresses {
		if !validTLSIngress(i) {
			continue
		}
		for _, tls := range i.Spec.TLS {
			secretName, secretNamespace := tls.SecretName, i.Namespace
			if secretName == "" && lc.DefaultTLSSecretName != "" {
				secretName = lc.DefaultTLSSecretName
				secretNamespace = lc.DefaultTLSSecretNamespace
			}
			secret, ok := secrets[metadata{name: secretName, namespace: secretNamespace}]
			if !ok {
				// no secret for this ingress yet, skip it
				continue
			}
			_, cert := secret.Data[v1.TLSCertKey]
			_, key := secret.Data[v1.TLSPrivateKeyKey]
			if !cert || !key {
				// missing cert or private key, skip it
				continue
			}
			fc := listener.FilterChain{
				FilterChainMatch: &listener.FilterChainMatch{
					SniDomains: tls.Hosts,
				},
				TlsContext: tlscontext(secret, "h2", "http/1.1"),
				Filters:    filters,
			}
			if lc.UseProxyProto {
				fc.UseProxyProto = &types.BoolValue{Value: true}
			}
			l.FilterChains = append(l.FilterChains, fc)
		}
	}

	switch len(l.FilterChains) {
	case 0:
		// no tls ingresses registered, remove the listener
		return nil, []string{l.Name}
	default:
		// at least one tls ingress registered, refresh listener
		return []*v2.Listener{l}, nil
	}
}

// httpsAddress returns the port for the HTTPS (TLS)
// listener or DEFAULT_HTTPS_LISTENER_ADDRESS if not configured.
func (lc *ListenerCache) httpsAddress() string {
	if lc.HTTPSAddress != "" {
		return lc.HTTPSAddress
	}
	return DEFAULT_HTTPS_LISTENER_ADDRESS
}

// httpsPort returns the port for the HTTPS (TLS) listener
// or DEFAULT_HTTPS_LISTENER_PORT if not configured.
func (lc *ListenerCache) httpsPort() uint32 {
	if lc.HTTPSPort != 0 {
		return uint32(lc.HTTPSPort)
	}
	return DEFAULT_HTTPS_LISTENER_PORT
}

// validTLSIngress returns true if this is a valid ssl ingress object.
// ingresses are invalid if they contain annotations, or are missing information
// which excludes them from the ingress_https listener.
func validTLSIngress(i *v1beta1.Ingress) bool {
	if len(i.Spec.TLS) == 0 {
		// this ingress does not use TLS, skip it
		return false
	}
	return true
}

func socketaddress(address string, port uint32) core.Address {
	return core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Protocol: core.TCP,
				Address:  address,
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

func tlscontext(secret *v1.Secret, alpnprotos ...string) *auth.DownstreamTlsContext {
	return &auth.DownstreamTlsContext{
		CommonTlsContext: &auth.CommonTlsContext{
			TlsParams: &auth.TlsParameters{
				// Envoy 1.5.0 defaults to TLS 1.0.
				TlsMinimumProtocolVersion: auth.TlsParameters_TLSv1_1,
			},
			TlsCertificates: []*auth.TlsCertificate{{
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: secret.Data[v1.TLSCertKey],
					},
				},
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: secret.Data[v1.TLSPrivateKeyKey],
					},
				},
			}},
			AlpnProtocols: alpnprotos,
		},
	}
}

func httpfilter(routename string, enabletracing bool) listener.Filter {
	filter := listener.Filter{
		Name: httpFilter,
		Config: &types.Struct{
			Fields: map[string]*types.Value{
				"stat_prefix": sv(routename),
				"rds": st(map[string]*types.Value{
					"route_config_name": sv(routename),
					"config_source": st(map[string]*types.Value{
						"api_config_source": st(map[string]*types.Value{
							"api_type": sv("GRPC"),
							"cluster_names": lv(
								sv("contour"),
							),
							"grpc_services": lv(
								st(map[string]*types.Value{
									"envoy_grpc": st(map[string]*types.Value{
										"cluster_name": sv("contour"),
									}),
								}),
							),
						}),
					}),
				}),
				"http_filters": lv(
					st(map[string]*types.Value{
						"name": sv(router),
					}),
				),
				"use_remote_address": bv(true), // TODO(jbeda) should this ever be false?
				"access_log": lv(
					st(map[string]*types.Value{
						"name": sv(accessLog),
						"config": st(map[string]*types.Value{
							"path": sv("/dev/stdout"),
						}),
					}),
				),
			},
		},
	}
	if enabletracing {
		filter.Config.Fields["tracing"] = st(map[string]*types.Value{
			"operation_name": sv("egress"),
		})
	}
	return filter
}

func filterchain(useproxy bool, filters ...listener.Filter) listener.FilterChain {
	fc := listener.FilterChain{
		Filters: filters,
	}
	if useproxy {
		fc.UseProxyProto = &types.BoolValue{Value: true}
	}
	return fc
}

func sv(s string) *types.Value {
	return &types.Value{Kind: &types.Value_StringValue{StringValue: s}}
}

func bv(b bool) *types.Value {
	return &types.Value{Kind: &types.Value_BoolValue{BoolValue: b}}
}

func st(m map[string]*types.Value) *types.Value {
	return &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{Fields: m}}}
}
func lv(v ...*types.Value) *types.Value {
	return &types.Value{Kind: &types.Value_ListValue{ListValue: &types.ListValue{Values: v}}}
}
