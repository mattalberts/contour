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
