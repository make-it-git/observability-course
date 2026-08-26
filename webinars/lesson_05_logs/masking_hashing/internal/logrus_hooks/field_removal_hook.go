package logrus_hooks

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

// FieldRemoverHook is a custom Logrus hook that removes specified fields from log entries.
type FieldRemoverHook struct {
	fieldsToRemove []string       // List of field names to remove
	levels         []logrus.Level // Log levels to apply the hook to.  If nil, applies to all.
}

// NewFieldRemoverHook creates a new FieldRemoverHook.
func NewFieldRemoverHook(fieldsToRemove []string, levels []logrus.Level) *FieldRemoverHook {
	return &FieldRemoverHook{
		fieldsToRemove: fieldsToRemove,
		levels:         levels,
	}
}

// Levels defines the log levels this hook will be applied to.
func (hook *FieldRemoverHook) Levels() []logrus.Level {
	if hook.Levels == nil {
		return logrus.AllLevels // Apply to all levels by default
	}
	return hook.levels
}

// Fire is called when a log entry matches the hook's levels.
func (hook *FieldRemoverHook) Fire(entry *logrus.Entry) error {
	for _, field := range hook.fieldsToRemove {
		fmt.Printf("Removing %s field\n", field) // Only for demo purposes
		delete(entry.Data, field)                // Remove the field from the entry's data.
	}
	return nil
}
