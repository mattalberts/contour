package dag

import (
	"time"
)

// RouteOptions defines optional route defaults
type RouteOptions struct {
	IdleTimeout    time.Duration
	MaxGrpcTimeout time.Duration
}
