package main

import (
	"errors"
	"fmt"
	"net/rpc"
	"strconv"
	"time"

	"github.com/v1Flows/runner/pkg/alerts"
	"github.com/v1Flows/runner/pkg/executions"
	"github.com/v1Flows/runner/pkg/flows"
	"github.com/v1Flows/runner/pkg/plugins"

	"github.com/v1Flows/alertFlow/services/backend/pkg/models"

	"github.com/hashicorp/go-plugin"
)

type Receiver struct {
	Receiver string `json:"receiver"`
}

// Plugin is an implementation of the Plugin interface
type Plugin struct{}

func (p *Plugin) ExecuteTask(request plugins.ExecuteTaskRequest) (plugins.Response, error) {
	flowID := ""
	alertID := ""
	logData := false

	if val, ok := request.Args["FlowID"]; ok {
		flowID = val
	}

	if val, ok := request.Args["AlertID"]; ok {
		alertID = val
	}

	if val, ok := request.Args["LogData"]; ok {
		logData, _ = strconv.ParseBool(val)
	}

	if request.Step.Action.Params != nil && flowID == "" && alertID == "" {
		for _, param := range request.Step.Action.Params {
			if param.Key == "LogData" {
				logData, _ = strconv.ParseBool(param.Value)
			}
			if param.Key == "FlowID" {
				flowID = param.Value
			}
			if param.Key == "AlertID" {
				alertID = param.Value
			}
		}
	}

	if flowID == "" || alertID == "" {
		_ = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
			ID:         request.Step.ID,
			Messages:   []string{"FlowID and AlertID are required"},
			Status:     "error",
			FinishedAt: time.Now(),
		})

		return plugins.Response{
			Success: false,
		}, errors.New("flowid and alertid are required")
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
		}, err
	}

	// Get Flow Data
	flow, err := flows.GetFlowData(request.Config, flowID, request.Execution.ID.String())
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
			}, err
		}

		return plugins.Response{
			Success: false,
		}, err
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:       request.Step.ID,
		Messages: []string{"Flow Data collected"},
	})
	if err != nil {
		return plugins.Response{
			Success: false,
		}, err
	}

	// Get Alert Data
	alert, err := alerts.GetData(request.Config, alertID)
	if err != nil {
		err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
			ID: request.Step.ID,
			Messages: []string{
				"Failed to get Alert Data",
				err.Error(),
			},
			Status:     "error",
			FinishedAt: time.Now(),
		})
		if err != nil {
			return plugins.Response{
				Success: false,
			}, err
		}

		return plugins.Response{
			Success: false,
		}, err
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:       request.Step.ID,
		Messages: []string{"Alert Data collected"},
	})
	if err != nil {
		return plugins.Response{
			Success: false,
		}, err
	}

	finalMessages := []string{}

	if logData {
		finalMessages = append(finalMessages, "Data collection completed")
		finalMessages = append(finalMessages, "Flow Data:")
		finalMessages = append(finalMessages, fmt.Sprintf("%v", flow))
		finalMessages = append(finalMessages, "Alert Data:")
		finalMessages = append(finalMessages, fmt.Sprintf("%v", alert))
	} else {
		finalMessages = append(finalMessages, "Data collection completed")
	}

	err = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID:         request.Step.ID,
		Messages:   finalMessages,
		Status:     "success",
		FinishedAt: time.Now(),
	})
	if err != nil {
		return plugins.Response{
			Success: false,
		}, err
	}

	return plugins.Response{
		Flow:    &flow,
		Alert:   &alert,
		Success: true,
	}, nil
}

func (p *Plugin) HandleAlert(request plugins.AlertHandlerRequest) (plugins.Response, error) {
	return plugins.Response{
		Success: false,
	}, errors.New("not implemented")
}

func (p *Plugin) Info() (models.Plugins, error) {
	var plugin = models.Plugins{
		Name:    "Collect Data",
		Type:    "action",
		Version: "1.1.2",
		Author:  "JustNZ",
		Actions: models.Actions{
			Name:        "Collect Data",
			Description: "Collects Flow and Alert data from AlertFlow",
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
					Key:         "AlertID",
					Type:        "text",
					Default:     "00000000-0000-0000-0000-00000000",
					Required:    true,
					Description: "The Alert ID to collect data from",
				},
			},
		},
		Endpoints: models.AlertEndpoints{},
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

func (s *PluginRPCServer) HandleAlert(request plugins.AlertHandlerRequest, resp *plugins.Response) error {
	result, err := s.Impl.HandleAlert(request)
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
			"plugin": &PluginServer{Impl: &Plugin{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
