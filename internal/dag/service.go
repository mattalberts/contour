package dag

import (
	"time"
)

// ServiceOptions defines optional service defaults
type ServiceOptions struct {
	ConnectTimeout                time.Duration
	MaxConnections                uint32
	MaxPendingRequests            uint32
	MaxRequests                   uint32
	MaxRetries                    uint32
	PerConnectionBufferLimitBytes uint32
	HTTP2ProtocolOptions          HTTP2ProtocolOptions
}

// ServiceLimits defines optional service defaults
type ServiceLimits struct {
	ConnectTimeout                time.Duration
	MaxConnections                uint32
	MaxPendingRequests            uint32
	MaxRequests                   uint32
	MaxRetries                    uint32
	PerConnectionBufferLimitBytes uint32
}

// HTTP2ProtocolOptions defines protocol options for http2 conntections
type HTTP2ProtocolOptions struct {
	AllowConnect                      bool
	MaxConcurrentStreams              uint32
	InitialConnectionWindowSize       uint32
	InitialStreamWindowSize           uint32
	StreamErrorOnInvalidHTTPMessaging bool
}
