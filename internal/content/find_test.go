package content_test

import (
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/content"
)

func TestBundleLookupByID(t *testing.T) {
	bundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if lesson, ok := bundle.LessonByID("lpic1.103.1.lesson.shell-sequences"); !ok || lesson.ID == "" {
		t.Fatalf("lesson lookup = %#v, %v", lesson, ok)
	}
	if question, ok := bundle.QuestionByID("lpic1.103.1.q.sequence-and"); !ok || question.ID == "" {
		t.Fatalf("question lookup = %#v, %v", question, ok)
	}
	if _, ok := bundle.LessonByID("missing"); ok {
		t.Fatal("missing lesson unexpectedly found")
	}
}
