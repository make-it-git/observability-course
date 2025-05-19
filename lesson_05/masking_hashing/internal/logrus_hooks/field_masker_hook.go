package logrus_hooks

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"hash/fnv"
)

// FieldHasherHook is a custom Logrus hook that masks specified fields in log entries.
type FieldHasherHook struct {
	fieldsToHash []string       // List of field names to mask
	levels       []logrus.Level // Log levels to apply the hook to.  If nil, applies to all.
}

// NewFieldHasherHook creates a new FieldHasherHook.
func NewFieldHasherHook(fieldsToMask []string, levels []logrus.Level) *FieldHasherHook {
	return &FieldHasherHook{
		fieldsToHash: fieldsToMask,
		levels:       levels,
	}
}

// Levels defines the log levels this hook will be applied to.
func (hook *FieldHasherHook) Levels() []logrus.Level {
	if hook.Levels == nil {
		return logrus.AllLevels // Apply to all levels by default
	}
	return hook.levels
}

// Fire is called when a log entry matches the hook's levels.
func (hook *FieldHasherHook) Fire(entry *logrus.Entry) error {
	for _, field := range hook.fieldsToHash {
		fmt.Printf("Hashing %s field\n", field)                           // Only for demo purposes
		entry.Data[fmt.Sprintf("%s_original", field)] = entry.Data[field] // Only for demo purposes
		hash := fnv.New64a()
		hash.Write([]byte(entry.Data[field].(string)))
		entry.Data[field] = hash.Sum64()
	}
	return nil
}
