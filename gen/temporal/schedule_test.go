package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const scheduleModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	item def Order { attribute id : String; }
	@Activity { taskQueue = "q"; }
	action def Charge { in order : Order; }
	@Workflow { id = "Nightly"; taskQueue = "q"; }
	@Schedule { spec = "0 0 * * *"; jitter = "5m"; }
	action def Nightly {
		in order : Order;
		action c : Charge;
	}
}`

func buildScheduleModel(t *testing.T) *engine.Model {
	t.Helper()
	m := engine.New().AddFile("app.sysml", scheduleModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	return m
}

func TestScheduleShape(t *testing.T) {
	src, err := GenerateSchedules(buildScheduleModel(t), "gen")
	if err != nil {
		t.Fatalf("GenerateSchedules: %v", err)
	}
	n := norm(src)
	for _, want := range []string{
		"func CreateNightlySchedule(ctx context.Context, c client.Client) error",
		"c.ScheduleClient().Create(ctx, client.ScheduleOptions{",
		`ID: "Nightly-schedule"`,
		`CronExpressions: []string{"0 0 * * *"}`,
		`Jitter: mustDuration("5m")`,
		"&client.ScheduleWorkflowAction{",
		"Workflow: NightlyWorkflow",
		`TaskQueue: "q"`,
		"func mustDuration(s string) time.Duration",
	} {
		if !strings.Contains(n, want) {
			t.Errorf("generated schedule missing %q\n---\n%s", want, src)
		}
	}
}

// TestScheduleCompiles builds the schedule alongside the workflows and
// activities it references, against the Temporal SDK.
func TestScheduleCompiles(t *testing.T) {
	m := buildScheduleModel(t)
	acts, err := GenerateActivities(m, "gen")
	if err != nil {
		t.Fatalf("GenerateActivities: %v", err)
	}
	wf, err := GenerateWorkflows(m, "gen")
	if err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}
	sched, err := GenerateSchedules(m, "gen")
	if err != nil {
		t.Fatalf("GenerateSchedules: %v", err)
	}
	compileFiles(t, map[string]string{
		"activities.go": acts,
		"workflows.go":  wf,
		"schedule.go":   sched,
	})
}

// TestNoJitter omits the jitter field and the mustDuration helper when no
// schedule declares jitter.
func TestNoJitter(t *testing.T) {
	src := `package App {
	import TemporalProfile::*;
	@Workflow { id = "W"; taskQueue = "q"; }
	@Schedule { spec = "@daily"; }
	action def W;
}`
	m := engine.New().AddFile("app.sysml", src).Build()
	out, err := GenerateSchedules(m, "gen")
	if err != nil {
		t.Fatalf("GenerateSchedules: %v", err)
	}
	if strings.Contains(out, "Jitter") || strings.Contains(out, "mustDuration") {
		t.Errorf("expected no jitter/mustDuration:\n%s", out)
	}
	if !strings.Contains(norm(out), `CronExpressions: []string{"@daily"}`) {
		t.Errorf("missing cron spec:\n%s", out)
	}
}

// TestNoSchedules emits no schedule functions for a model without @Schedule.
func TestNoSchedules(t *testing.T) {
	src := `package App {
	import TemporalProfile::*;
	@Workflow { id = "W"; taskQueue = "q"; }
	action def W;
}`
	m := engine.New().AddFile("app.sysml", src).Build()
	out, err := GenerateSchedules(m, "gen")
	if err != nil {
		t.Fatalf("GenerateSchedules: %v", err)
	}
	if strings.Contains(out, "func Create") {
		t.Errorf("unexpected schedule function:\n%s", out)
	}
}
