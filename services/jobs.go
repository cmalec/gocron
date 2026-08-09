package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/glebarez/go-sqlite"
	"github.com/labstack/echo/v5"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"

	"github.com/flohoss/gocron/config"
	"github.com/flohoss/gocron/internal/commands"
	"github.com/flohoss/gocron/internal/events"
	"github.com/flohoss/gocron/internal/healthcheck"
	"github.com/flohoss/gocron/internal/scheduler"
	"github.com/flohoss/gocron/services/jobs"
)

const (
	DATE_FORMAT = "2006-01-02 15:04:05"
)

func formatTime(startTime int64) string {
	startSeconds := startTime / 1000
	t := time.Unix(startSeconds, 0).Local()
	return t.Format(DATE_FORMAT)
}

var (
	lastTimestamp int64
	mu            sync.Mutex
)

func generateUniqueTimestamp() int64 {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UnixMilli()
	if now <= lastTimestamp {
		lastTimestamp++
	} else {
		lastTimestamp = now
	}
	return lastTimestamp
}

type JobView struct {
	config.Job
	NextRunUnix int64     `json:"next_run_unix"`
	NextRun     string    `json:"next_run"`
	SuccessRate float64   `json:"success_rate"`
	RunCount    int64     `json:"run_count"`
	Runs        []RunView `json:"runs"`
}

type DayStat struct {
	Day       string `json:"day"`
	Succeeded int64  `json:"succeeded"`
	Failed    int64  `json:"failed"`
	Total     int64  `json:"total"`
}

type ActivityRun struct {
	ID            int64  `json:"id"`
	JobName       string `json:"job_name"`
	JobSlug       string `json:"job_slug"`
	StatusID      int64  `json:"status_id"`
	StartTimeUnix int64  `json:"start_time_unix"`
	StartTime     string `json:"start_time"`
	Duration      string `json:"duration"`
}

type RunView struct {
	ID            int64                      `json:"id"`
	JobName       string                     `json:"job_name"`
	StatusID      int64                      `json:"status_id"`
	StartTimeUnix int64                      `json:"start_time_unix"`
	StartTime     string                     `json:"start_time"`
	EndTime       string                     `json:"end_time"`
	Duration      string                     `json:"duration"`
	Logs          []jobs.ListLogsByRunIDsRow `json:"logs"`
}

func NewJobService() (*JobService, error) {
	queries, err := setupSQLite()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	js := &JobService{Queries: queries, jobCtx: ctx, jobCancel: cancel}
	queries.StopRunning(context.Background())
	js.loadJobStates()
	js.setupJobs()
	js.setupViperWatcher()

	return js, nil
}

func (js *JobService) loadJobStates() {
	states, err := js.Queries.GetJobStates(context.Background())
	if err != nil {
		slog.Error("Failed to load job states", "error", err)
		return
	}
	disabled := make(map[string]bool, len(states))
	for _, state := range states {
		disabled[state.JobSlug] = state.Disabled != 0
	}
	config.SetDisabledStates(disabled)
}

func (js *JobService) setupJobs() {
	// stop any previous running scheduler without waiting for in-flight jobs
	if js.Scheduler != nil {
		_ = js.Scheduler.Stop()
	}

	js.Scheduler = scheduler.New()
	for _, job := range config.GetJobs() {
		js.scheduleJob(job)
	}

	if config.GetDeleteRunsAfterDays() > 0 {
		js.Scheduler.Add("0 0 * * *", func() {
			js.Queries.DeleteOldRuns(context.Background(), time.Now().AddDate(0, 0, -int(config.GetDeleteRunsAfterDays())).UnixMilli())
		})
	}
	// delete any orphaned runs and states inside the db for cleanup
	deleteOrphanedRuns(js.Queries)
}

func (js *JobService) scheduleJob(job config.Job) {
	if job.DisableCron {
		return
	}
	cronExpr := config.GetJobsCron(&job)
	if cronExpr == "" {
		return
	}
	j := job
	if err := js.Scheduler.AddJob(job.Slug, cronExpr, func() { js.runScheduled(j) }); err != nil {
		slog.Error("Failed to schedule job", "job", job.Name, "cron", cronExpr, "error", err)
	}
}

func (js *JobService) runScheduled(job config.Job) {
	current := config.GetJobByName(job.Slug)
	if current == nil || current.Disabled {
		return
	}
	js.ExecuteJobs([]config.Job{*current})
}

func (js *JobService) setupViperWatcher() {
	var (
		mu    sync.Mutex
		timer *time.Timer
	)

	debounce := func(d time.Duration, fn func()) {
		mu.Lock()
		defer mu.Unlock()

		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(d, fn)
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		debounce(2*time.Second, func() {
			slog.Info("Config changed, reloading jobs")
			err := config.ValidateAndLoadConfig(viper.GetViper())
			if err != nil {
				slog.Error("Failed to reload configuration, keeping old settings", "error", err)
				return
			}
			slog.Info("Config reloaded successfully, reloading jobs")
			js.setupJobs()
			js.Events.SendJobEvent(js.IsIdle(), nil, js.ListJobs())
		})
	})

	viper.WatchConfig()
}

type JobService struct {
	Queries    *jobs.Queries
	Scheduler  *scheduler.Scheduler
	Events     *events.Event
	jobCtx     context.Context
	jobCancel  context.CancelFunc
	jobMu      sync.Mutex
	jobRunning bool
}

func (js *JobService) SetEvents(e *events.Event) {
	js.Events = e
}

func (js *JobService) GetQueries() *jobs.Queries {
	return js.Queries
}

func (js *JobService) GetParser() *cron.Parser {
	return js.Scheduler.GetParser()
}

func (js *JobService) GetHandler() echo.HandlerFunc {
	return js.Events.GetHandler()
}

func (js *JobService) IsIdle() bool {
	res, _ := js.Queries.IsIdle(context.Background())
	return res == 1
}

func (js *JobService) Shutdown() {
	if js.Scheduler != nil {
		<-js.Scheduler.Stop().Done()
	}
	js.jobCancel()
	deadline := time.Now().Add(8 * time.Second)
	for {
		js.jobMu.Lock()
		running := js.jobRunning
		js.jobMu.Unlock()
		if !running || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	js.Queries.StopRunning(context.Background())
}

func deleteOrphanedRuns(queries *jobs.Queries) {
	slugs := []string{}
	for _, job := range config.GetJobs() {
		slugs = append(slugs, job.Slug)
	}
	queries.DeleteObsoleteRuns(context.Background(), slugs)
	queries.DeleteObsoleteJobStates(context.Background(), slugs)
}

func (js *JobService) nextRun(job config.Job) (int64, string) {
	if job.DisableCron {
		return 0, ""
	}
	cronExpr := config.GetJobsCron(&job)
	if cronExpr == "" {
		return 0, ""
	}
	schedule, err := js.Scheduler.GetParser().Parse(cronExpr)
	if err != nil {
		return 0, ""
	}
	next := schedule.Next(time.Now())
	return next.UnixMilli(), next.Local().Format(DATE_FORMAT)
}

func (js *JobService) SetJobDisabled(name string, disabled bool) error {
	job := config.GetJobByName(name)
	if job == nil {
		return fmt.Errorf("job %q not found", name)
	}
	if err := config.SetJobDisabled(job.Name, disabled); err != nil {
		return err
	}
	var disabledInt int64
	if disabled {
		disabledInt = 1
	}
	if err := js.Queries.SetJobState(context.Background(), jobs.SetJobStateParams{JobSlug: job.Slug, Disabled: disabledInt}); err != nil {
		slog.Error("Failed to persist job state", "job", job.Name, "error", err)
	}
	js.Events.SendJobEvent(js.IsIdle(), nil, js.ListJobs())
	return nil
}

func (js *JobService) GetActivity(limit int64) []ActivityRun {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := js.Queries.GetRecentRuns(context.Background(), limit)
	if err != nil {
		slog.Error(err.Error())
		return []ActivityRun{}
	}
	result := make([]ActivityRun, 0, len(rows))
	for _, row := range rows {
		var duration time.Duration
		if row.EndTime.Valid {
			duration = time.Duration(row.EndTime.Int64-row.StartTime) * time.Millisecond
		}
		result = append(result, ActivityRun{
			ID:            row.ID,
			JobName:       row.JobName,
			JobSlug:       row.JobSlug,
			StatusID:      row.StatusID,
			StartTimeUnix: row.StartTime,
			StartTime:     formatTime(row.StartTime),
			Duration:      duration.Truncate(time.Second).String(),
		})
	}
	return result
}

func (js *JobService) GetDailyStats(name string, days int64) ([]DayStat, error) {
	job := config.GetJobByName(name)
	if job == nil {
		return nil, fmt.Errorf("job %q not found", name)
	}
	if days <= 0 {
		days = 90
	}
	since := time.Now().AddDate(0, 0, -int(days)).UnixMilli()
	rows, err := js.Queries.GetDailyRunStats(context.Background(), jobs.GetDailyRunStatsParams{JobSlug: job.Slug, StartTime: since})
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats for job %s: %w", name, err)
	}
	stats := make([]DayStat, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, DayStat{
			Day:       row.Day,
			Succeeded: row.Succeeded,
			Failed:    row.Failed,
			Total:     row.Total,
		})
	}
	return stats, nil
}

func (js *JobService) ExecuteJobs(jobs []config.Job) {
	js.jobMu.Lock()
	if js.jobRunning || !js.IsIdle() {
		js.jobMu.Unlock()
		return
	}
	js.jobRunning = true
	js.jobMu.Unlock()
	defer func() {
		js.jobMu.Lock()
		js.jobRunning = false
		js.jobMu.Unlock()
	}()

	if len(jobs) == 0 {
		jobs = config.GetJobs()
	}
	healthcheck.SendStart()
	for _, job := range jobs {
		if len(jobs) > 0 && job.Disabled {
			continue
		}
		js.ExecuteJob(&job)
	}
	healthcheck.SendEnd()
}

func (js *JobService) ExecuteJob(job *config.Job) {
	ctx := js.jobCtx

	run, err := js.startRun(ctx, job.Name, job.Slug)
	if err != nil {
		slog.Error(err.Error())
		healthcheck.SendFailure()
		return
	}

	// Key storage for log
	keys := []string{}
	envs := config.GetEnvsForJob(job)
	for _, key := range envs.Order {
		os.Setenv(key, os.ExpandEnv(envs.Data[key]))
		keys = append(keys, key)
	}

	js.writeLog(ctx, run, Debug, fmt.Sprintf("Setting environment variables: %s", strings.Join(keys, ", ")))

	failed := false
	timeout := config.GetTimeoutForJob(job)
	retries := config.GetRetriesForJob(job)
	for _, command := range config.GetCommandsForJob(job) {
		severity := Debug
		js.writeLog(ctx, run, Debug, fmt.Sprintf("Executing command: %s", command))

		var out string
		var err error
		for attempt := 0; attempt <= retries; attempt++ {
			if attempt > 0 {
				js.writeLog(ctx, run, Debug, fmt.Sprintf("Retrying command (attempt %d/%d): %s", attempt, retries, command))
			}
			out, err = commands.ExecuteCommandWithContext(ctx, command, timeout)
			if err == nil {
				break
			}
		}

		severity = Info
		if err != nil {
			if errors.Is(err, context.Canceled) {
				run.StatusID = Canceled.Int64()
				js.endRun(context.Background(), run)
				return
			}
			severity = Error
			healthcheck.SendFailure()
		}
		js.writeLog(ctx, run, severity, out)
		if err != nil {
			js.writeLog(ctx, run, Error, err.Error())
			failed = true
			run.StatusID = Stopped.Int64()
			if !job.DisableFailFast {
				break
			}
		} else if !failed {
			run.StatusID = Finished.Int64()
		}
	}

	for _, key := range envs.Order {
		os.Unsetenv(key)
	}

	js.endRun(ctx, run)
}

func (js *JobService) ListJobs() []JobView {
	jobs := config.GetJobs()

	runs, err := js.Queries.GetThreeRunsPerJobName(context.Background())
	if err != nil {
		slog.Error(err.Error())
		return []JobView{}
	}

	runsByJob := make(map[string][]RunView)
	for _, run := range runs {
		endTime := ""
		var duration time.Duration
		if run.EndTime.Valid {
			endTime = formatTime(run.EndTime.Int64)
			duration = time.Duration(run.EndTime.Int64-run.StartTime) * time.Millisecond
		}
		runsByJob[run.JobName] = append(runsByJob[run.JobName],
			RunView{
				ID:            run.ID,
				StatusID:      run.StatusID,
				StartTime:     formatTime(run.StartTime),
				StartTimeUnix: run.StartTime,
				EndTime:       endTime,
				Duration:      duration.Truncate(time.Second).String(),
			})
	}

	rates := make(map[string][2]int64)
	rateRows, err := js.Queries.GetJobSuccessRates(context.Background())
	if err != nil {
		slog.Error(err.Error())
	}
	for _, row := range rateRows {
		rates[row.JobSlug] = [2]int64{row.Succeeded, row.Total}
	}

	result := make([]JobView, 0, len(jobs))
	for _, job := range jobs {
		nextUnix, nextFormatted := js.nextRun(job)
		rate := rates[job.Slug]
		succeeded, total := rate[0], rate[1]
		var successRate float64
		if total > 0 {
			successRate = float64(succeeded) / float64(total) * 100
		}
		result = append(result, JobView{
			Job: config.Job{
				Name:        job.Name,
				Slug:        job.Slug,
				Cron:        config.GetJobsCron(&job),
				DisableCron: job.DisableCron,
				Disabled:    job.Disabled,
			},
			NextRunUnix: nextUnix,
			NextRun:     nextFormatted,
			SuccessRate: successRate,
			RunCount:    total,
			Runs:        runsByJob[job.Name],
		})
	}

	return result
}

func (js *JobService) ListRuns(name string, limit int64, includeLogs bool) ([]RunView, error) {
	runs, err := js.Queries.GetRuns(context.Background(), jobs.GetRunsParams{JobSlug: name, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("failed to get runs for job %s: %w", name, err)
	}

	if len(runs) == 0 {
		return []RunView{}, nil
	}

	logsByRun := make(map[int64][]jobs.ListLogsByRunIDsRow)
	if includeLogs {
		runIDs := make([]int64, 0, len(runs))
		for _, run := range runs {
			runIDs = append(runIDs, run.ID)
		}
		allLogs, err := js.Queries.ListLogsByRunIDs(context.Background(), runIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get logs for runs: %w", err)
		}
		for _, log := range allLogs {
			logsByRun[log.RunID] = append(logsByRun[log.RunID], log)
		}
	}

	result := make([]RunView, 0, len(runs))
	for _, run := range runs {
		endTime := ""
		var duration time.Duration
		if run.EndTime.Valid {
			endTime = formatTime(run.EndTime.Int64)
			duration = time.Duration(run.EndTime.Int64-run.StartTime) * time.Millisecond
		}

		result = append(result, RunView{
			ID:            run.ID,
			JobName:       run.JobName,
			StatusID:      run.StatusID,
			StartTime:     formatTime(run.StartTime),
			StartTimeUnix: run.StartTime,
			EndTime:       endTime,
			Duration:      duration.Truncate(time.Second).String(),
			Logs:          logsByRun[run.ID],
		})
	}

	return result, nil
}

func (js *JobService) startRun(ctx context.Context, jobName string, jobSlug string) (*jobs.Run, error) {
	run, err := js.Queries.CreateRun(ctx, jobs.CreateRunParams{
		JobName:   jobName,
		JobSlug:   jobSlug,
		StatusID:  Running.Int64(),
		StartTime: time.Now().UnixMilli(),
	})
	if err != nil {
		return nil, err
	}
	js.Events.SendJobEvent(true, js.getLatestRun(ctx, &run), nil)
	return &run, nil
}

func (js *JobService) endRun(ctx context.Context, run *jobs.Run) {
	_, err := js.Queries.UpdateRun(ctx, jobs.UpdateRunParams{
		StatusID: run.StatusID,
		EndTime:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:       run.ID,
	})
	if err != nil {
		slog.Error(err.Error())
		return
	}
	js.Events.SendJobEvent(true, js.getLatestRun(ctx, run), nil)
}

func (js *JobService) writeLog(ctx context.Context, run *jobs.Run, severity Severity, message string) {
	_, err := js.Queries.CreateLog(ctx, jobs.CreateLogParams{
		CreatedAt:  generateUniqueTimestamp(),
		RunID:      run.ID,
		SeverityID: int64(severity),
		Message:    message,
	})
	if err != nil {
		slog.Error(err.Error())
		return
	}
	js.Events.SendJobEvent(false, js.getLatestRun(ctx, run), nil)
}

func (js *JobService) getLatestRun(ctx context.Context, run *jobs.Run) *RunView {
	runs, err := js.Queries.GetRuns(ctx, jobs.GetRunsParams{JobSlug: run.JobSlug, Limit: 1})
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	if len(runs) == 0 {
		slog.Warn("No new run found")
		return nil
	}
	r := runs[0]
	logs, err := js.Queries.ListLogsByRunIDs(context.Background(), []int64{r.ID})
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	endTime := ""
	var duration time.Duration
	if r.EndTime.Valid {
		endTime = formatTime(r.EndTime.Int64)
		duration = time.Duration(r.EndTime.Int64-r.StartTime) * time.Millisecond
	}
	return &RunView{
		ID:            r.ID,
		JobName:       r.JobName,
		StatusID:      r.StatusID,
		StartTime:     formatTime(r.StartTime),
		StartTimeUnix: r.StartTime,
		EndTime:       endTime,
		Duration:      duration.Truncate(time.Second).String(),
		Logs:          logs,
	}
}
