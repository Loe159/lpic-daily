package content

func (bundle *Bundle) LessonByID(id string) (Lesson, bool) {
	if bundle == nil {
		return Lesson{}, false
	}
	for _, lesson := range bundle.Lessons {
		if lesson.ID == id {
			return lesson, true
		}
	}
	return Lesson{}, false
}

func (bundle *Bundle) QuestionByID(id string) (Question, bool) {
	if bundle == nil {
		return Question{}, false
	}
	for _, question := range bundle.Questions {
		if question.ID == id {
			return question, true
		}
	}
	return Question{}, false
}
