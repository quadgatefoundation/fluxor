package core

import (
	"testing"
)

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()

	if logger == nil {
		t.Error("NewDefaultLogger() should not return nil")
	}

	// Test that logger methods don't panic
	logger.Error("test error")
	logger.Errorf("test error: %s", "message")
	logger.Warn("test warning")
	logger.Warnf("test warning: %s", "message")
	logger.Info("test info")
	logger.Infof("test info: %s", "message")
	logger.Debug("test debug")
	logger.Debugf("test debug: %s", "message")
}
