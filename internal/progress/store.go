package progress

import (
	"context"

	"github.com/Loe159/lpic-daily/internal/learning"
)

type EvidenceStore interface {
	AppendEvidence(context.Context, learning.EvidenceEvent) error
	EvidenceForConcept(context.Context, string) ([]learning.EvidenceEvent, error)
	Close() error
}
