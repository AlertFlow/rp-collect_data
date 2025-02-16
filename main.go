package main

import (
	"fmt"
	"net/rpc"
	"strconv"
	"time"

	"github.com/AlertFlow/runner/pkg/executions"
	"github.com/AlertFlow/runner/pkg/flows"
	"github.com/AlertFlow/runner/pkg/payloads"
	"github.com/AlertFlow/runner/pkg/plugins"

	"github.com/v1Flows/alertFlow/services/backend/pkg/models"

	"github.com/hashicorp/go-plugin"
)

type Receiver struct {
	Receiver string `json:"receiver"`
}

// CollectDataActionPlugin is an implementation of the Plugin interface
type CollectDataActionPlugin struct{}

func (p *CollectDataActionPlugin) ExecuteTask(request plugins.ExecuteTaskRequest) (plugins.Response, error) {
	flowID := ""
	payloadID := ""
	logData := false

	if val, ok := request.Args["FlowID"]; ok {
		flowID = val
	}

	if val, ok := request.Args["PayloadID"]; ok {
		payloadID = val
	}

	if val, ok := request.Args["LogData"]; ok {
		logData, _ = strconv.ParseBool(val)
	}

	if request.Step.Action.Params != nil && flowID == "" && payloadID == "" {
		for _, param := range request.Step.Action.Params {
			if param.Key == "LogData" {
				logData, _ = strconv.ParseBool(param.Value)
			}
			if param.Key == "FlowID" {
				flowID = param.Value
			}
			if param.Key == "PayloadID" {
				payloadID = param.Value
			}
		}
	}

	if flowID == "" || payloadID == "" {
		return plugins.Response{
			Success: false,
			Error:   "FlowID and PayloadID are required",
		}, nil
	}

	err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:        request.Step.ID,
		Messages:  []string{"Collecting data from AlertFlow"},
		Status:    "running",
		StartedAt: time.Now(),
	})
	if err != nil {
		return plugins.Response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Get Flow Data
	flow, err := flows.GetFlowData(request.Config, flowID)
	if err != nil {
		err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
			ID: request.Step.ID,
			Messages: []string{
				"Failed to get Flow Data",
				err.Error(),
			},
			Status:     "error",
			FinishedAt: time.Now(),
		})
		if err != nil {
			return plugins.Response{
				Success: false,
				Error:   err.Error(),
			}, err
		}

		return plugins.Response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:       request.Step.ID,
		Messages: []string{"Flow Data collected"},
	})
	if err != nil {
		return plugins.Response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Get Payload Data
	payload, err := payloads.GetData(request.Config, payloadID)
	if err != nil {
		err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
			ID: request.Step.ID,
			Messages: []string{
				"Failed to get Payload Data",
				err.Error(),
			},
			Status:     "error",
			FinishedAt: time.Now(),
		})
		if err != nil {
			return plugins.Response{
				Success: false,
				Error:   err.Error(),
			}, err
		}

		return plugins.Response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:       request.Step.ID,
		Messages: []string{"Payload Data collected"},
	})
	if err != nil {
		return plugins.Response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	finalMessages := []string{}

	if logData {
		finalMessages = append(finalMessages, "Data collection completed")
		finalMessages = append(finalMessages, "Flow Data:")
		finalMessages = append(finalMessages, fmt.Sprintf("%v", flow))
		finalMessages = append(finalMessages, "Payload Data:")
		finalMessages = append(finalMessages, fmt.Sprintf("%v", payload))
	} else {
		finalMessages = append(finalMessages, "Data collection completed")
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:         request.Step.ID,
		Messages:   finalMessages,
		Status:     "finished",
		FinishedAt: time.Now(),
	})
	if err != nil {
		return plugins.Response{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return plugins.Response{
		Data: map[string]interface{}{
			"flow":    flow,
			"payload": payload,
		},
		Success: true,
	}, nil
}

func (p *CollectDataActionPlugin) HandlePayload(request plugins.PayloadHandlerRequest) (plugins.Response, error) {
	return plugins.Response{
		Success: false,
		Error:   "Not implemented",
	}, nil
}

func (p *CollectDataActionPlugin) Info() (models.Plugins, error) {
	var plugin = models.Plugins{
		Name:    "Collect Data",
		Type:    "action",
		Version: "1.1.0",
		Author:  "JustNZ",
		Actions: models.Actions{
			Name:        "Collect Data",
			Description: "Collects Flow and Payload data from AlertFlow",
			Plugin:      "collect_data",
			Icon:        "solar:inbox-archive-linear",
			Category:    "Data",
			Params: []models.Params{
				{
					Key:         "LogData",
					Type:        "boolean",
					Default:     "false",
					Required:    false,
					Description: "Show collected data in the output messages",
				},
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
		Endpoints: models.PayloadEndpoints{},
	}

	return plugin, nil
}

// PluginRPCServer is the RPC server for Plugin
type PluginRPCServer struct {
	Impl plugins.Plugin
}

func (s *PluginRPCServer) ExecuteTask(request plugins.ExecuteTaskRequest, resp *plugins.Response) error {
	result, err := s.Impl.ExecuteTask(request)
	*resp = result
	return err
}

func (s *PluginRPCServer) HandlePayload(request plugins.PayloadHandlerRequest, resp *plugins.Response) error {
	result, err := s.Impl.HandlePayload(request)
	*resp = result
	return err
}

func (s *PluginRPCServer) Info(args interface{}, resp *models.Plugins) error {
	result, err := s.Impl.Info()
	*resp = result
	return err
}

// PluginServer is the implementation of plugin.Plugin interface
type PluginServer struct {
	Impl plugins.Plugin
}

func (p *PluginServer) Server(*plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

func (p *PluginServer) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &plugins.PluginRPC{Client: c}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "PLUGIN_MAGIC_COOKIE",
			MagicCookieValue: "hello",
		},
		Plugins: map[string]plugin.Plugin{
			"plugin": &PluginServer{Impl: &CollectDataActionPlugin{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
