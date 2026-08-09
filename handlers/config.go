package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/flohoss/gocron/config"
)

type JobConfigInput struct {
	Name        string   `json:"name" minLength:"1" maxLength:"255" doc:"job name"`
	Cron        string   `json:"cron" doc:"cron expression (empty with disable_cron for manual-only)"`
	DisableCron bool     `json:"disable_cron" doc:"run only when triggered manually"`
	Commands    []string `json:"commands" minLength:"1" doc:"commands to execute in order"`
}

func validateCronExpression(expr string, disableCron bool) error {
	if disableCron || strings.TrimSpace(expr) == "" {
		return nil
	}
	if err := config.ValidateCronExpression(expr); err != nil {
		return huma.Error400BadRequest("Invalid cron expression", err)
	}
	return nil
}

func (jh *JobHandler) updateJobConfigOperation() huma.Operation {
	return huma.Operation{
		OperationID: "update-job-config",
		Method:      http.MethodPut,
		Path:        "/api/config/jobs/{name}",
		Summary:     "Update job config",
		Description: "Update a job's schedule and commands, written back to the config file.",
		Tags:        []string{"Config"},
	}
}

func (jh *JobHandler) updateJobConfigHandler(ctx context.Context, input *struct {
	Name string `path:"name" maxLength:"255" doc:"job name"`
	Body JobConfigInput
}) (*struct{}, error) {
	job := config.GetJobByName(input.Name)
	if job == nil {
		return nil, huma.Error404NotFound("Job not found")
	}
	if err := validateCronExpression(input.Body.Cron, input.Body.DisableCron); err != nil {
		return nil, err
	}
	if len(input.Body.Commands) == 0 {
		return nil, huma.Error400BadRequest("At least one command is required")
	}
	err := config.UpdateJobInConfig(job.Name, config.JobWriteInput{
		Name:        input.Body.Name,
		Cron:        input.Body.Cron,
		DisableCron: input.Body.DisableCron,
		Commands:    input.Body.Commands,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to write config", err)
	}
	return nil, nil
}

func (jh *JobHandler) createJobConfigOperation() huma.Operation {
	return huma.Operation{
		OperationID: "create-job-config",
		Method:      http.MethodPost,
		Path:        "/api/config/jobs",
		Summary:     "Create job",
		Description: "Add a new job, written back to the config file.",
		Tags:        []string{"Config"},
	}
}

func (jh *JobHandler) createJobConfigHandler(ctx context.Context, input *struct {
	Body JobConfigInput
}) (*struct{}, error) {
	if config.GetJobByName(input.Body.Name) != nil {
		return nil, huma.Error409Conflict("A job with this name already exists")
	}
	if err := validateCronExpression(input.Body.Cron, input.Body.DisableCron); err != nil {
		return nil, err
	}
	if len(input.Body.Commands) == 0 {
		return nil, huma.Error400BadRequest("At least one command is required")
	}
	err := config.AddJobToConfig(config.JobWriteInput{
		Name:        input.Body.Name,
		Cron:        input.Body.Cron,
		DisableCron: input.Body.DisableCron,
		Commands:    input.Body.Commands,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to write config", err)
	}
	return nil, nil
}

func (jh *JobHandler) deleteJobConfigOperation() huma.Operation {
	return huma.Operation{
		OperationID: "delete-job-config",
		Method:      http.MethodDelete,
		Path:        "/api/config/jobs/{name}",
		Summary:     "Delete job",
		Description: "Remove a job from the config file.",
		Tags:        []string{"Config"},
	}
}

func (jh *JobHandler) deleteJobConfigHandler(ctx context.Context, input *struct {
	Name string `path:"name" maxLength:"255" doc:"job name"`
}) (*struct{}, error) {
	job := config.GetJobByName(input.Name)
	if job == nil {
		return nil, huma.Error404NotFound("Job not found")
	}
	if err := config.DeleteJobFromConfig(job.Name); err != nil {
		return nil, huma.Error500InternalServerError("Failed to write config", err)
	}
	return nil, nil
}
