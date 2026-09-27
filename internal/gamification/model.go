package gamification

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type EventType string

const (
	EventLessonCompleted     EventType = "lesson-completed"
	EventQuestionPassed      EventType = "question-passed"
	EventQuestionFailed      EventType = "question-failed"
	EventLabPassed           EventType = "lab-passed"
	EventAchievementUnlocked EventType = "achievement-unlocked"
)

type Event struct {
	EventID    string
	OccurredAt time.Time
	Type       EventType
	Amount     int
	Metadata   map[string]string
}

func (event Event) Validate() error {
	if event.EventID == "" {
		return errors.New("event ID is required")
	}
	if event.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	switch event.Type {
	case EventLessonCompleted, EventQuestionPassed, EventQuestionFailed, EventLabPassed, EventAchievementUnlocked:
	default:
		return fmt.Errorf("unsupported event type %q", event.Type)
	}
	if event.Amount < 0 {
		return errors.New("amount cannot be negative")
	}
	return nil
}

type Achievement struct {
	ID             string
	TitleFR        string
	DescriptionFR  string
	XPReward       int
	AffectsMastery bool
	Metric         string
	Threshold      int
}

var BuiltinAchievements = []Achievement{
	{
		ID:             "first-steps",
		TitleFR:        "Premiers pas",
		DescriptionFR:  "Terminer une première activité LPIC Daily.",
		XPReward:       20,
		AffectsMastery: false,
		Metric:         "activities-completed",
		Threshold:      1,
	},
	{
		ID:             "first-lab",
		TitleFR:        "Les mains dans le système",
		DescriptionFR:  "Réussir un premier lab pratique.",
		XPReward:       40,
		AffectsMastery: false,
		Metric:         "labs-passed",
		Threshold:      1,
	},
	{
		ID:             "streak-3",
		TitleFR:        "Régularité",
		DescriptionFR:  "Étudier trois jours consécutifs.",
		XPReward:       50,
		AffectsMastery: false,
		Metric:         "streak-days",
		Threshold:      3,
	},
}

type Snapshot struct {
	XP                   int
	CurrentStreakDays    int
	ActivitiesCompleted  int
	LabsPassed           int
	UnlockedAchievements []string
	LastActivityAt       time.Time
}

func Project(events []Event, location *time.Location, now time.Time) (Snapshot, error) {
	if location == nil {
		location = time.Local
	}
	if now.IsZero() {
		return Snapshot{}, errors.New("now is required")
	}

	ordered := append([]Event(nil), events...)
	slices.SortFunc(ordered, func(a, b Event) int {
		switch {
		case a.OccurredAt.Before(b.OccurredAt):
			return -1
		case a.OccurredAt.After(b.OccurredAt):
			return 1
		case a.EventID < b.EventID:
			return -1
		case a.EventID > b.EventID:
			return 1
		default:
			return 0
		}
	})

	snapshot := Snapshot{}
	activeDays := map[string]struct{}{}
	unlocked := map[string]struct{}{}

	for _, event := range ordered {
		if err := event.Validate(); err != nil {
			return Snapshot{}, fmt.Errorf("event %s: %w", event.EventID, err)
		}
		snapshot.XP += event.Amount

		switch event.Type {
		case EventLessonCompleted, EventQuestionPassed, EventQuestionFailed, EventLabPassed:
			snapshot.ActivitiesCompleted++
			day := event.OccurredAt.In(location).Format("2006-01-02")
			activeDays[day] = struct{}{}
			if event.OccurredAt.After(snapshot.LastActivityAt) {
				snapshot.LastActivityAt = event.OccurredAt
			}
		case EventAchievementUnlocked:
			if id := event.Metadata["achievement_id"]; id != "" {
				unlocked[id] = struct{}{}
			}
		}
		if event.Type == EventLabPassed {
			snapshot.LabsPassed++
		}
	}

	cursor := midnight(now.In(location))
	for {
		key := cursor.Format("2006-01-02")
		if _, exists := activeDays[key]; !exists {
			break
		}
		snapshot.CurrentStreakDays++
		cursor = cursor.AddDate(0, 0, -1)
	}

	for id := range unlocked {
		snapshot.UnlockedAchievements = append(snapshot.UnlockedAchievements, id)
	}
	slices.Sort(snapshot.UnlockedAchievements)
	return snapshot, nil
}

func NewlyUnlocked(snapshot Snapshot) []Achievement {
	unlocked := make(map[string]struct{}, len(snapshot.UnlockedAchievements))
	for _, id := range snapshot.UnlockedAchievements {
		unlocked[id] = struct{}{}
	}

	var result []Achievement
	for _, achievement := range BuiltinAchievements {
		if _, exists := unlocked[achievement.ID]; exists {
			continue
		}
		value := 0
		switch achievement.Metric {
		case "activities-completed":
			value = snapshot.ActivitiesCompleted
		case "labs-passed":
			value = snapshot.LabsPassed
		case "streak-days":
			value = snapshot.CurrentStreakDays
		default:
			continue
		}
		if value >= achievement.Threshold {
			result = append(result, achievement)
		}
	}
	return result
}

func midnight(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func XPFor(eventType EventType) int {
	switch eventType {
	case EventLessonCompleted:
		return 10
	case EventQuestionPassed:
		return 15
	case EventQuestionFailed:
		return 3
	case EventLabPassed:
		return 50
	default:
		return 0
	}
}
