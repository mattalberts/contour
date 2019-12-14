package dag

// ServiceOptions defines optional service defaults
type ServiceOptions struct {
	MaxConnections     uint32
	MaxPendingRequests uint32
	MaxRequests        uint32
	MaxRetries         uint32
}

// ServiceLimits defines optional service defaults
type ServiceLimits struct {
	MaxConnections     uint32
	MaxPendingRequests uint32
	MaxRequests        uint32
	MaxRetries         uint32
}
