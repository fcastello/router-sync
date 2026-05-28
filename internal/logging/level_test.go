package logging

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetAndGetLevel(t *testing.T) {
	Init(logrus.WarnLevel, "test")
	assert.Equal(t, "warning", GetLevelName())
	assert.Equal(t, "test", ServiceID())

	SetLevel(logrus.DebugLevel)
	assert.Equal(t, logrus.DebugLevel, GetLevel())
	assert.Equal(t, logrus.DebugLevel, logrus.GetLevel())
}

func TestParseLevel(t *testing.T) {
	level, err := ParseLevel("DEBUG")
	assert.NoError(t, err)
	assert.Equal(t, logrus.DebugLevel, level)

	_, err = ParseLevel("nope")
	assert.Error(t, err)
}
