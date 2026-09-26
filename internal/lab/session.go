package lab

import "time"

type Session struct {
	ID          string
	ContainerID string
	Spec        Spec
	CreatedAt   time.Time
	ExpiresAt   time.Time
}
