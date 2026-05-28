package logging

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	mu           sync.RWMutex
	currentLevel = logrus.WarnLevel
	serviceID    = ""
)

// SupportedLevels lists valid runtime log levels (lowest to highest verbosity).
var SupportedLevels = []logrus.Level{
	logrus.TraceLevel,
	logrus.DebugLevel,
	logrus.InfoLevel,
	logrus.WarnLevel,
	logrus.ErrorLevel,
	logrus.FatalLevel,
	logrus.PanicLevel,
}

// Init sets the global logrus level and identifies this process by service ID.
// service IDs follow the convention "api" or "agent.<hostname>".
func Init(level logrus.Level, service string) {
	mu.Lock()
	serviceID = service
	mu.Unlock()
	SetLevel(level)
}

// ServiceID returns the log service identifier for this process.
func ServiceID() string {
	mu.RLock()
	defer mu.RUnlock()
	return serviceID
}

// GetLevel returns the current logrus level.
func GetLevel() logrus.Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// GetLevelName returns the current level as a lowercase string (e.g. "info").
func GetLevelName() string {
	return GetLevel().String()
}

// SetLevel updates logrus and the stored runtime level.
func SetLevel(level logrus.Level) {
	mu.Lock()
	currentLevel = level
	mu.Unlock()
	logrus.SetLevel(level)
}

// ParseLevel parses a level name (case-insensitive).
func ParseLevel(name string) (logrus.Level, error) {
	level, err := logrus.ParseLevel(name)
	if err != nil {
		return 0, fmt.Errorf("invalid log level %q: use trace, debug, info, warn, error, fatal, or panic", name)
	}
	return level, nil
}

// LevelNames returns supported level names for API responses.
func LevelNames() []string {
	names := make([]string, len(SupportedLevels))
	for i, l := range SupportedLevels {
		names[i] = l.String()
	}
	return names
}
