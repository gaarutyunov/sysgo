package temporal

import (
	"strconv"

	"github.com/gaarutyunov/sysgo/engine"
)

// Workflow is an @Workflow-annotated action and its Temporal configuration.
type Workflow struct {
	Element   engine.Element
	ID        string
	TaskQueue string
}

// Name returns the workflow's element name.
func (w Workflow) Name() string { return w.Element.Name() }

// RetryPolicy holds an activity's @RetryPolicy values, read as text.
type RetryPolicy struct {
	MaxAttempts        string
	InitialInterval    string
	BackoffCoefficient string
	MaxInterval        string
}

// Timeout holds an activity's @Timeout values, read as text.
type Timeout struct {
	StartToClose    string
	ScheduleToClose string
	Heartbeat       string
}

// Activity is an @Activity-annotated action and its Temporal configuration.
type Activity struct {
	Element    engine.Element
	TaskQueue  string
	Idempotent bool
	Retry      *RetryPolicy
	Timeout    *Timeout
}

// Name returns the activity's element name.
func (a Activity) Name() string { return a.Element.Name() }

// Workflows returns every @Workflow-annotated action in the model's user
// packages, with its resolved configuration.
func Workflows(m *engine.Model) []Workflow {
	var out []Workflow
	walkUser(m, func(e engine.Element) {
		if meta, ok := e.Metadata("Workflow"); ok {
			out = append(out, Workflow{
				Element:   e,
				ID:        val(meta, "id"),
				TaskQueue: val(meta, "taskQueue"),
			})
		}
	})
	return out
}

// Activities returns every @Activity-annotated action in the model's user
// packages, with its resolved configuration (retry, timeout, idempotency).
func Activities(m *engine.Model) []Activity {
	var out []Activity
	walkUser(m, func(e engine.Element) {
		meta, ok := e.Metadata("Activity")
		if !ok {
			return
		}
		a := Activity{Element: e, TaskQueue: val(meta, "taskQueue")}
		if _, ok := e.Metadata("Idempotent"); ok {
			a.Idempotent = true
		}
		if rp, ok := e.Metadata("RetryPolicy"); ok {
			a.Retry = &RetryPolicy{
				MaxAttempts:        val(rp, "maxAttempts"),
				InitialInterval:    val(rp, "initialInterval"),
				BackoffCoefficient: val(rp, "backoffCoefficient"),
				MaxInterval:        val(rp, "maxInterval"),
			}
		}
		if to, ok := e.Metadata("Timeout"); ok {
			a.Timeout = &Timeout{
				StartToClose:    val(to, "startToClose"),
				ScheduleToClose: val(to, "scheduleToClose"),
				Heartbeat:       val(to, "heartbeat"),
			}
		}
		out = append(out, a)
	})
	return out
}

func val(meta engine.Metadata, key string) string {
	if v, ok := meta.Value(key); ok {
		return unquote(v)
	}
	return ""
}

// unquote strips surrounding quotes from a string-literal metadata value; a
// non-quoted value (e.g. a number) is returned as-is.
func unquote(s string) string {
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	return s
}

// walkUser visits every element in the model's user packages (excluding the
// bundled standard library and metadata profiles) in pre-order.
func walkUser(m *engine.Model, fn func(engine.Element)) {
	var walk func(engine.Element)
	walk = func(e engine.Element) {
		fn(e)
		for _, c := range e.Children() {
			walk(c)
		}
	}
	for _, top := range m.Root().Children() {
		if isBundledPackage(top.Name()) {
			continue
		}
		walk(top)
	}
}

func isBundledPackage(name string) bool {
	switch name {
	case "ScalarValues", "Base", "RESTProfile", "TemporalProfile":
		return true
	default:
		return false
	}
}
