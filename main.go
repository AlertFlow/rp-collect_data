package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlertFlow/runner/pkg/executions"
	"github.com/AlertFlow/runner/pkg/flows"
	"github.com/AlertFlow/runner/pkg/models"
	"github.com/AlertFlow/runner/pkg/payloads"
	"github.com/AlertFlow/runner/pkg/protocol"
)

func main() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		os.Exit(0)
	}()

	// Process requests
	for {
		var req protocol.Request
		if err := decoder.Decode(&req); err != nil {
			os.Exit(1)
		}

		// Handle the request
		resp := handle(req)

		if err := encoder.Encode(resp); err != nil {
			os.Exit(1)
		}
	}
}

func Details() models.Plugin {
	plugin := models.Plugin{
		Name:    "Collect Data",
		Type:    "action",
		Version: "1.0.6",
		Author:  "JustNZ",
		Action: models.ActionDetails{
			ID:          "collect_data",
			Name:        "Collect Data",
			Description: "Collects Flow and Payload data from AlertFlow",
			Icon:        "solar:inbox-archive-linear",
			Category:    "Data",
			Params: []models.Param{
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
			},
		},
	}

	return plugin
}

func execute(execution models.Execution, step models.ExecutionSteps, action models.Actions) (outputData map[string]interface{}, success bool, err error) {
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

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionID:       action.ID.String(),
		ActionMessages: []string{"Collecting data from AlertFlow"},
		Pending:        false,
		Running:        true,
		StartedAt:      time.Now(),
	})
	if err != nil {
		return nil, false, err
	}

	// Get Flow Data
	flow, err := flows.GetFlowData(flowID)
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
			return nil, false, err
		}

		return nil, false, err
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{"Flow Data collected"},
	})
	if err != nil {
		return nil, false, err
	}

	// Get Payload Data
	payload, err := payloads.GetData(payloadID)
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
			return nil, false, err
		}

		return nil, false, err
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{"Payload Data collected"},
	})
	if err != nil {
		return nil, false, err
	}

	err = executions.UpdateStep(execution.ID.String(), models.ExecutionSteps{
		ID:             step.ID,
		ActionMessages: []string{"Data collection completed"},
		Running:        false,
		Finished:       true,
		FinishedAt:     time.Now(),
	})
	if err != nil {
		return nil, false, err
	}

	return map[string]interface{}{"flow": flow, "payload": payload}, true, nil
}

func handle(req protocol.Request) protocol.Response {
	switch req.Action {
	case "details":
		return protocol.Response{
			Success: true,
			Plugin:  Details(),
		}

	case "execute":
		outputData, success, err := execute(req.Data["execution"].(models.Execution), req.Data["step"].(models.ExecutionSteps), req.Data["action"].(models.Actions))
		if err != nil {
			return protocol.Response{
				Success: false,
				Error:   err.Error(),
			}
		}

		return protocol.Response{
			Success: success,
			Data:    outputData,
		}

	default:
		return protocol.Response{
			Success: false,
			Error:   "unknown action",
		}
	}
}
