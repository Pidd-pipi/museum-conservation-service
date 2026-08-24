package main

import "testing"

func TestScheduleVerifyingFlowCompletes(t *testing.T) {
	schedule := newOpsScheduleService()
	if _, err := schedule.Plan("rec-flow-1", OpsPriorityHigh); err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := schedule.Move("rec-flow-1", OpsScheduleInspecting); err != nil {
		t.Fatalf("to inspecting: %v", err)
	}
	if _, err := schedule.Move("rec-flow-1", OpsScheduleVerifying); err != nil {
		t.Fatalf("to verifying: %v", err)
	}
	if _, err := schedule.Move("rec-flow-1", OpsScheduleTreating); err != nil {
		t.Fatalf("verifying must move to treating, got: %v", err)
	}
	if _, err := schedule.Move("rec-flow-1", OpsScheduleDone); err != nil {
		t.Fatalf("to done: %v", err)
	}
	item, err := schedule.Get("rec-flow-1")
	if err != nil || item.Status != OpsScheduleDone {
		t.Fatalf("schedule should end done, got %+v err=%v", item, err)
	}
}

func TestScheduleStatusSummaryIncludesVerifying(t *testing.T) {
	schedule := newOpsScheduleService()
	if _, err := schedule.Plan("rec-sum-1", OpsPriorityNormal); err != nil {
		t.Fatalf("plan 1: %v", err)
	}
	if _, err := schedule.Plan("rec-sum-2", OpsPriorityNormal); err != nil {
		t.Fatalf("plan 2: %v", err)
	}
	if _, err := schedule.Plan("rec-sum-3", OpsPriorityNormal); err != nil {
		t.Fatalf("plan 3: %v", err)
	}
	_, _ = schedule.Move("rec-sum-1", OpsScheduleInspecting)
	_, _ = schedule.Move("rec-sum-1", OpsScheduleVerifying)
	_, _ = schedule.Move("rec-sum-2", OpsScheduleInspecting)
	_, _ = schedule.Move("rec-sum-3", OpsScheduleInspecting)

	summary := schedule.StatusSummary()
	if summary["in_progress"] != 3 {
		t.Fatalf("in_progress must include verifying, got %+v", summary)
	}
	if summary["planned"] != 0 {
		t.Fatalf("verifying must not be counted as planned, got %+v", summary)
	}
}

func TestAlertEngineVerifyingState(t *testing.T) {
	schedule := newOpsScheduleService()
	if _, err := schedule.Plan("rec-alert-1", OpsPriorityNormal); err != nil {
		t.Fatalf("plan: %v", err)
	}
	_, _ = schedule.Move("rec-alert-1", OpsScheduleInspecting)
	_, _ = schedule.Move("rec-alert-1", OpsScheduleVerifying)

	engine := newOpsAlertEngine(schedule)
	alerts := engine.Evaluate()
	found := false
	for _, alert := range alerts {
		if alert.RecordID == "rec-alert-1" {
			found = true
			if alert.Code != "SCHED-VERIFYING" || alert.Level != "warn" {
				t.Fatalf("verifying item should produce a verification-in-progress warning, got %+v", alert)
			}
		}
	}
	if !found {
		t.Fatalf("no alert for verifying item: %+v", alerts)
	}
}
