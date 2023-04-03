package terraform

import (
	"encoding/json"
	"strings"
)

// TODO: can be used during deploy time too

type Resource struct {
	Name string `json:"resource_name"`
	Type string `json:"resource_type"`
}

type Change struct {
	Resource *Resource `json:"resource"`
	Action   string    `json:"action"`
}

type Summary struct {
	Add    int `json:"add"`
	Change int `json:"change"`
	Remove int `json:"remove"`
}

type Event struct {
	Change  *Change  `json:"change"`
	Summary *Summary `json:"changes"`
	Type    string   `json:"type"`
}

type Plan struct {
	ChangeSummary  *Event
	PlannedChanges []Event
}

func NewPlan() *Plan {
	return &Plan{
		PlannedChanges: make([]Event, 0),
	}
}

func (p *Plan) AddEvent(s string) error {
	event := &Event{}
	if len(s) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(s), event)
	if err != nil {
		return err
	}
	switch event.Type {
	case "planned_change":
		p.PlannedChanges = append(p.PlannedChanges, *event)
	case "change_summary":
		p.ChangeSummary = event
	}
	return nil
}

func (c *Change) String() string {
	result := strings.Builder{}
	result.WriteString(c.Action + " ")
	result.WriteString(c.Resource.Type + " ")
	result.WriteString(c.Resource.Name)

	return result.String()
}
