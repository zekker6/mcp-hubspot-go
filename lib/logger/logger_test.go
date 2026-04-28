package logger

import (
	"testing"

	"go.uber.org/zap"
)

func resetLogger() {
	logger = nil
}

func TestInitStopNoPanic(t *testing.T) {
	resetLogger()
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if logger == nil {
		t.Fatal("logger is nil after Init")
	}
	Stop()
}

func TestEmitFunctionsNoPanic(t *testing.T) {
	resetLogger()
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Stop()
	Info("info-msg", zap.String("k", "v"))
	Error("error-msg", zap.String("k", "v"))
	Warn("warn-msg", zap.String("k", "v"))
	Debug("debug-msg", zap.String("k", "v"))
}

func TestWithReturnsLogger(t *testing.T) {
	resetLogger()
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Stop()
	l := With(zap.String("k", "v"))
	if l == nil {
		t.Fatal("With() returned nil")
	}
}

func TestLevelSwitch(t *testing.T) {
	cases := []string{"debug", "info", "warn", "error"}
	for _, lvl := range cases {
		t.Run(lvl, func(t *testing.T) {
			orig := *logLevel
			*logLevel = lvl
			resetLogger()
			t.Cleanup(func() {
				*logLevel = orig
				resetLogger()
			})
			if err := Init(); err != nil {
				t.Fatalf("Init: %v", err)
			}
			if logger == nil {
				t.Fatal("logger is nil after Init")
			}
		})
	}
}

func TestUnknownLevelReturnsError(t *testing.T) {
	orig := *logLevel
	*logLevel = "garbage"
	resetLogger()
	t.Cleanup(func() {
		*logLevel = orig
		resetLogger()
	})

	err := Init()
	if err == nil {
		t.Fatal("Init did not return error on unknown level")
	}
	if logger != nil {
		t.Fatal("logger should remain nil when Init fails")
	}
}

func TestInitIdempotent(t *testing.T) {
	resetLogger()
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	first := logger
	if err := Init(); err != nil {
		t.Fatalf("Init second call: %v", err)
	}
	if logger != first {
		t.Fatal("Init() re-created logger when already set")
	}
}

func TestEmitBeforeInitIsNoop(t *testing.T) {
	resetLogger()
	// Must not panic even though logger is nil.
	Info("before-init")
	Error("before-init")
	Warn("before-init")
	Debug("before-init")
	if l := With(zap.String("k", "v")); l == nil {
		t.Fatal("With before Init returned nil instead of nop")
	}
}
