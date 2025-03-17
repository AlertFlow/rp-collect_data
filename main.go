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

	"github.com/v1Flows/shared-library/pkg/models"

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

	if flowID == "" || (request.Platform == "alertflow" && alertID == "") {
		_ = executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
			ID: request.Step.ID,
			Messages: []models.Message{
				{
					Title: "Collecting Data",
					Lines: []string{"FlowID and AlertID are required"},
				},
			},
			Status:     "error",
			FinishedAt: time.Now(),
		})

		return plugins.Response{
			Success: false,
		}, errors.New("flowid and alertid are required")
	}

	err := executions.UpdateStep(request.Config, request.Execution.ID.String(), models.ExecutionSteps{
		ID: request.Step.ID,
		Messages: []models.Message{
			{
				Title: "Collecting Data",
				Lines: []string{"Collecting data from " + request.Platform},
			},
		},
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
			Messages: []models.Message{
				{
					Title: "Collecting Data",
					Lines: []string{"Failed to get Flow Data", err.Error()},
				},
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
		ID: request.Step.ID,
		Messages: []models.Message{
			{
				Title: "Collecting Data",
				Lines: []string{"Flow Data collected"},
			},
		},
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
			Messages: []models.Message{
				{
					Title: "Collecting Data",
					Lines: []string{"Failed to get Alert Data", err.Error()},
				},
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
		ID: request.Step.ID,
		Messages: []models.Message{
			{
				Title: "Collecting Data",
				Lines: []string{"Alert Data collected"},
			},
		},
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
		ID: request.Step.ID,
		Messages: []models.Message{
			{
				Title: "Collecting Data",
				Lines: finalMessages,
			},
		},
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

func (p *Plugin) EndpointRequest(request plugins.EndpointRequest) (plugins.Response, error) {
	return plugins.Response{
		Success: false,
	}, errors.New("not implemented")
}

func (p *Plugin) Info() (models.Plugin, error) {
	var plugin = models.Plugin{
		Name:    "Collect Data",
		Type:    "action",
		Version: "1.1.3",
		Author:  "JustNZ",
		Action: models.Action{
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
					Required:    false,
					Description: "The Alert ID to collect data from. Required for AlertFlow platform",
				},
			},
		},
		Endpoint: models.Endpoint{},
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

func (s *PluginRPCServer) EndpointRequest(request plugins.EndpointRequest, resp *plugins.Response) error {
	result, err := s.Impl.EndpointRequest(request)
	*resp = result
	return err
}

func (s *PluginRPCServer) Info(args interface{}, resp *models.Plugin) error {
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
