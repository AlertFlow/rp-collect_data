package main

import (
	"encoding/json"
	"time"

	"gitlab.justlab.xyz/alertflow-public/runner/pkg/executions"
	"gitlab.justlab.xyz/alertflow-public/runner/pkg/flows"
	"gitlab.justlab.xyz/alertflow-public/runner/pkg/models"
	"gitlab.justlab.xyz/alertflow-public/runner/pkg/payloads"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type CollectDataPlugin struct{}

func (p *CollectDataPlugin) Init() models.Plugin {
	return models.Plugin{
		Name:    "Collect Data",
		Type:    "action",
		Version: "1.0.4",
		Creator: "JustNZ",
	}
}

func (p *CollectDataPlugin) Details() models.PluginDetails {
	params := []models.Param{
		{
			Key:         "FlowID",
			Type:        "text",
			Default:     "00000000-0000-0000-0000-00000000",
			Required:    true,
			Description: "The Flow ID to collect data from",
		},
		{
			Key:         "PayloadID",
			Type:        "text",
			Default:     "00000000-0000-0000-0000-00000000",
			Required:    true,
			Description: "The Payload ID to collect data from",
		},
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Error(err)
	}

	return models.PluginDetails{
		Action: models.ActionDetails{
			Name:        "Collect Data",
			Description: "Collects Flow and Payload data from AlertFlow",
			Icon:        "solar:inbox-archive-linear",
			Type:        "collect_data",
			Category:    "Data",
			Function:    p.Execute,
			IsHidden:    true,
			Params:      json.RawMessage(paramsJSON),
		},
	}
}

func (p *CollectDataPlugin) Execute(execution models.Execution, flow models.Flows, payload models.Payload, steps []models.ExecutionSteps, step models.ExecutionSteps, action models.Actions) (data map[string]interface{}, finished bool, canceled bool, no_pattern_match bool, failed bool) {
	flowID := ""
	payloadID := ""

	if action.Params == nil {
		flowID = execution.FlowID
		payloadID = execution.PayloadID
	} else {
		for _, param := range action.Params {
			if param.Key == "FlowID" {
				flowID = param.Value
			}
			if param.Key == "PayloadID" {
				payloadID = param.Value
			}
		}
	}

	err := executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionID:       action.ID.String(),
		ActionMessages: []string{"Collecting data from AlertFlow"},
		Pending:        false,
		Running:        true,
		StartedAt:      time.Now(),
	})
	if err != nil {
		log.Error("Error updating step: ", err)
	}

	// Get Flow Data
	flow, err = flows.GetFlowData(flowID)
	if err != nil {
		err := executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
			ID:             step.ID,
			ActionMessages: []string{"Failed to get Flow Data"},
			Error:          true,
			Finished:       true,
			Running:        false,
			FinishedAt:     time.Now(),
		})
		if err != nil {
			log.Error("Error updating step: ", err)
		}

		return nil, false, false, false, true
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{"Flow Data collected"},
	})
	if err != nil {
		log.Error("Error updating step: ", err)
	}

	// Get Payload Data
	payload, err = payloads.GetData(payloadID)
	if err != nil {
		err := executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
			ID:             step.ID,
			ActionMessages: []string{"Failed to get Payload Data"},
			Error:          true,
			Finished:       true,
			Running:        false,
			FinishedAt:     time.Now(),
		})
		if err != nil {
			log.Error("Error updating step: ", err)
		}

		return nil, false, false, false, true
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{"Payload Data collected"},
	})
	if err != nil {
		log.Error("Error updating step: ", err)
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{"Data collection completed"},
		Running:        false,
		Finished:       true,
		FinishedAt:     time.Now(),
	})
	if err != nil {
		log.Error("Error updating step: ", err)
	}

	return map[string]interface{}{"flow": flow, "payload": payload}, true, false, false, false
}

func (p *CollectDataPlugin) Handle(context *gin.Context) {}

var Plugin CollectDataPlugin
