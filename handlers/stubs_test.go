package handlers

import (
	"net/http"

	"github.com/flohoss/gocron/config"
	"github.com/flohoss/gocron/services"
	"github.com/flohoss/gocron/services/jobs"
	"github.com/labstack/echo/v5"
)

type stubJobService struct {
	queries *jobs.Queries
}

func (s *stubJobService) GetQueries() *jobs.Queries { return s.queries }

func (s *stubJobService) GetHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	}
}

func (s *stubJobService) IsIdle() bool { return true }

func (s *stubJobService) ExecuteJobs([]config.Job) {}

func (s *stubJobService) ExecuteJob(*config.Job) {}

func (s *stubJobService) ListJobs() []services.JobView { return nil }

func (s *stubJobService) ListRuns(string, int64, bool) ([]services.RunView, error) { return nil, nil }

func (s *stubJobService) SetJobDisabled(string, bool) error { return nil }

func (s *stubJobService) GetDailyStats(string, int64) ([]services.DayStat, error) { return nil, nil }

func (s *stubJobService) GetActivity(int64) []services.ActivityRun { return nil }

type stubCommandsService struct{}

func (s *stubCommandsService) ExecuteCommand(string) {}
