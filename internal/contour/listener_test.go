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

package contour

import (
	"testing"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/golang/protobuf/proto"
	ingressroutev1 "github.com/projectcontour/contour/apis/contour/v1beta1"
	projcontour "github.com/projectcontour/contour/apis/projectcontour/v1"
	"github.com/projectcontour/contour/internal/assert"
	"github.com/projectcontour/contour/internal/envoy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListenerCacheContents(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*v2.Listener
		want     []proto.Message
	}{
		"empty": {
			contents: nil,
			want:     nil,
		},
		"simple": {
			contents: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
			want: []proto.Message{
				&v2.Listener{
					Name:         ENVOY_HTTP_LISTENER,
					Address:      envoy.SocketAddress("0.0.0.0", 8080),
					FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var lc ListenerCache
			lc.Update(tc.contents)
			got := lc.Contents()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestListenerCacheQuery(t *testing.T) {
	tests := map[string]struct {
		contents map[string]*v2.Listener
		query    []string
		want     []proto.Message
	}{
		"exact match": {
			contents: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
			query: []string{ENVOY_HTTP_LISTENER},
			want: []proto.Message{
				&v2.Listener{
					Name:         ENVOY_HTTP_LISTENER,
					Address:      envoy.SocketAddress("0.0.0.0", 8080),
					FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				},
			},
		},
		"partial match": {
			contents: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
			query: []string{ENVOY_HTTP_LISTENER, "stats-listener"},
			want: []proto.Message{
				&v2.Listener{
					Name:         ENVOY_HTTP_LISTENER,
					Address:      envoy.SocketAddress("0.0.0.0", 8080),
					FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				},
			},
		},
		"no match": {
			contents: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
			query: []string{"stats-listener"},
			want:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var lc ListenerCache
			lc.Update(tc.contents)
			got := lc.Query(tc.query)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestListenerVisit(t *testing.T) {
	tests := map[string]struct {
		ListenerVisitorConfig
		objs []interface{}
		want map[string]*v2.Listener
	}{
		"nothing": {
			objs: nil,
			want: map[string]*v2.Listener{},
		},
		"one http only ingress": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: backend("kuard", 8080),
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
		},
		"one http only ingressroute": {
			objs: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							Fqdn: "www.example.com",
						},
						Routes: []ingressroutev1.Route{{
							Services: []ingressroutev1.Service{
								{
									Name: "backend",
									Port: 80,
								},
							},
						}},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     80,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
		},
		"simple ingress with secret": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
			}),
		},
		"multiple tls ingress with secrets should be sorted": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sortedsecond",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"sortedsecond.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "sortedsecond.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sortedfirst",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"sortedfirst.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "sortedfirst.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"sortedfirst.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}, {
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"sortedsecond.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
			}),
		},
		"simple ingress with missing secret": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "missing",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}),
		},
		"simple ingressroute with secret": {
			objs: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							Fqdn: "www.example.com",
							TLS: &projcontour.TLS{
								SecretName: "secret",
							},
						},
						Routes: []ingressroutev1.Route{
							{
								Services: []ingressroutev1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     80,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"www.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"ingress with allow-http: false": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.allow-http": "false",
						},
					},
					Spec: v1beta1.IngressSpec{
						Backend: backend("kuard", 8080),
					},
				},
			},
			want: map[string]*v2.Listener{},
		},
		"simple tls ingress with allow-http:false": {
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/ingress.allow-http": "false",
						},
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"www.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "www.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"www.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"http listener on non default port": { // issue 72
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPAddress:  "127.0.0.100",
				HTTPPort:     9100,
				HTTPSAddress: "127.0.0.200",
				HTTPSPort:    9200,
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("127.0.0.100", 9100),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("127.0.0.200", 9200),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
			}),
		},
		"use proxy proto": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				UseProxyProto: true,
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:    ENVOY_HTTP_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8080),
				ListenerFilters: envoy.ListenerFilters(
					envoy.ProxyProtocol(),
				),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				ListenerFilters: envoy.ListenerFilters(
					envoy.ProxyProtocol(),
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
			}),
		},
		"--envoy-http-access-log": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPAccessLog:  "/tmp/http_access.log",
				HTTPSAccessLog: "/tmp/https_access.log",
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress(DEFAULT_HTTP_LISTENER_ADDRESS, DEFAULT_HTTP_LISTENER_PORT),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy("/tmp/http_access.log"), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTPS_LISTENER_ADDRESS, DEFAULT_HTTPS_LISTENER_PORT),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy("/tmp/https_access.log"), envoy.HTTPConnectionOptions{})),
				}},
			}),
		},
		"tls-min-protocol-version from config": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				MinimumProtocolVersion: envoy_api_v2_auth.TlsParameters_TLSv1_3,
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_3, "h2", "http/1.1"),
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"tls-min-protocol-version from config overridden by annotation": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				MinimumProtocolVersion: envoy_api_v2_auth.TlsParameters_TLSv1_3,
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"projectcontour.io/tls-minimum-protocol-version": "1.2",
						},
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_3, "h2", "http/1.1"), // note, cannot downgrade from the configured version
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"tls-min-protocol-version from config overridden by legacy annotation": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				MinimumProtocolVersion: envoy_api_v2_auth.TlsParameters_TLSv1_3,
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
						Annotations: map[string]string{
							"contour.heptio.com/tls-minimum-protocol-version": "1.2",
						},
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_3, "h2", "http/1.1"), // note, cannot downgrade from the configured version
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"tls-min-protocol-version from config overridden by ingressroute": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				MinimumProtocolVersion: envoy_api_v2_auth.TlsParameters_TLSv1_3,
			},
			objs: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							Fqdn: "www.example.com",
							TLS: &projcontour.TLS{
								SecretName:             "secret",
								MinimumProtocolVersion: "1.2",
							},
						},
						Routes: []ingressroutev1.Route{
							{
								Services: []ingressroutev1.Service{
									{
										Name: "backend",
										Port: 80,
									},
								},
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     80,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:         ENVOY_HTTP_LISTENER,
				Address:      envoy.SocketAddress("0.0.0.0", 8080),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress("0.0.0.0", 8443),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"www.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_3, "h2", "http/1.1"), // note, cannot downgrade from the configured version
					Filters:    envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{})),
				}},
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
			}),
		},
		"--enable-trace": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPConnectionOptions: envoy.HTTPConnectionOptions{
					EnableTracing: true,
				},
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:    ENVOY_HTTP_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTP_LISTENER_ADDRESS, DEFAULT_HTTP_LISTENER_PORT),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{
					EnableTracing: true,
				})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTPS_LISTENER_ADDRESS, DEFAULT_HTTPS_LISTENER_PORT),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters: envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{
						EnableTracing: true,
					})),
				}},
			}),
		},
		"--idle-timeout": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPConnectionOptions: envoy.HTTPConnectionOptions{
					IdleTimeout: 1500 * time.Millisecond,
				},
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:    ENVOY_HTTP_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTP_LISTENER_ADDRESS, DEFAULT_HTTP_LISTENER_PORT),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{
					IdleTimeout: 1500 * time.Millisecond,
				})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTPS_LISTENER_ADDRESS, DEFAULT_HTTPS_LISTENER_PORT),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters: envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{
						IdleTimeout: 1500 * time.Millisecond,
					})),
				}},
			}),
		},
		"--request-timeout": {
			ListenerVisitorConfig: ListenerVisitorConfig{
				HTTPConnectionOptions: envoy.HTTPConnectionOptions{
					RequestTimeout: 1500 * time.Millisecond,
				},
			},
			objs: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							Hosts:      []string{"whatever.example.com"},
							SecretName: "secret",
						}},
						Rules: []v1beta1.IngressRule{{
							Host: "whatever.example.com",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{{
										Backend: *backend("kuard", 8080),
									}},
								},
							},
						}},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "default",
					},
					Type: "kubernetes.io/tls",
					Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name:     "http",
							Protocol: "TCP",
							Port:     8080,
						}},
					},
				},
			},
			want: listenermap(&v2.Listener{
				Name:    ENVOY_HTTP_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTP_LISTENER_ADDRESS, DEFAULT_HTTP_LISTENER_PORT),
				FilterChains: envoy.FilterChains(envoy.HTTPConnectionManager(ENVOY_HTTP_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{
					RequestTimeout: 1500 * time.Millisecond,
				})),
			}, &v2.Listener{
				Name:    ENVOY_HTTPS_LISTENER,
				Address: envoy.SocketAddress(DEFAULT_HTTPS_LISTENER_ADDRESS, DEFAULT_HTTPS_LISTENER_PORT),
				ListenerFilters: envoy.ListenerFilters(
					envoy.TLSInspector(),
				),
				FilterChains: []*envoy_api_v2_listener.FilterChain{{
					FilterChainMatch: &envoy_api_v2_listener.FilterChainMatch{
						ServerNames: []string{"whatever.example.com"},
					},
					TlsContext: tlscontext(envoy_api_v2_auth.TlsParameters_TLSv1_1, "h2", "http/1.1"),
					Filters: envoy.Filters(envoy.HTTPConnectionManager(ENVOY_HTTPS_LISTENER, envoy.FileAccessLogEnvoy(DEFAULT_HTTP_ACCESS_LOG), envoy.HTTPConnectionOptions{
						RequestTimeout: 1500 * time.Millisecond,
					})),
				}},
			}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			root := buildDAG(t, tc.objs...)
			got := visitListeners(root, &tc.ListenerVisitorConfig)
			assert.Equal(t, tc.want, got)
		})
	}
}

func tlscontext(tlsMinProtoVersion envoy_api_v2_auth.TlsParameters_TlsProtocol, alpnprotos ...string) *envoy_api_v2_auth.DownstreamTlsContext {
	return envoy.DownstreamTLSContext("default/secret/28337303ac", tlsMinProtoVersion, alpnprotos...)
}

func listenermap(listeners ...*v2.Listener) map[string]*v2.Listener {
	m := make(map[string]*v2.Listener)
	for _, l := range listeners {
		m[l.Name] = l
	}
	return m
}
