package dag

import (
	"time"
)

// RouteOptions defines optional route defaults
type RouteOptions struct {
	MaxGrpcTimeout time.Duration
}
