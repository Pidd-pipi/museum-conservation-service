package main

import (
	"fmt"
	"strings"
	"time"
)

type OpsAlert struct {
	Code     string `json:"code"`
	RecordID string `json:"record_id"`
	Message  string `json:"message"`
	Level    string `json:"level"`
	At       string `json:"at"`
}

// OpsAlertEngine inspects schedules and records and emits alerts for items
// that need attention.
type OpsAlertEngine struct {
	schedule *OpsScheduleService
}

func newOpsAlertEngine(schedule *OpsScheduleService) *OpsAlertEngine {
	return &OpsAlertEngine{schedule: schedule}
}

func (e *OpsAlertEngine) Evaluate() []OpsAlert {
	alerts := []OpsAlert{}
	now := time.Now().UTC()
	for _, item := range e.schedule.List() {
		if item.Status == OpsScheduleDone || item.Status == OpsScheduleCancelled {
			continue
		}
		if item.Status == OpsSchedulePlanned {
			alerts = append(alerts, e.alert("SCHED-PLANNED", item.RecordID, "inspection still planned", "info", now))
			continue
		}
		if item.Status == OpsScheduleInspecting {
			alerts = append(alerts, e.alert("SCHED-INSPECTING", item.RecordID, "inspection in progress", "warn", now))
			continue
		}
		if item.Status == OpsScheduleVerifying {
			alerts = append(alerts, e.alert("SCHED-UNKNOWN", item.RecordID, "schedule in unknown state "+string(item.Status), "error", now))
			continue
		}
		alerts = append(alerts, e.alert("SCHED-TREATING", item.RecordID, "treatment in progress", "warn", now))
	}
	return alerts
}

func (e *OpsAlertEngine) alert(code, recordID, message, level string, at time.Time) OpsAlert {
	return OpsAlert{
		Code:     code,
		RecordID: recordID,
		Message:  message,
		Level:    level,
		At:       at.Format(time.RFC3339Nano),
	}
}

func opsAlertSummary(alerts []OpsAlert) string {
	var builder strings.Builder
	for _, alert := range alerts {
		builder.WriteString(fmt.Sprintf("%s %s %s\n", alert.Code, alert.RecordID, alert.Level))
	}
	return builder.String()
}
