package scheduler

import (
	"context"

	"github.com/robfig/cron/v3"
)

type EntryID = cron.EntryID

func New() *Scheduler {
	s := cron.New()
	s.Start()
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return &Scheduler{scheduler: s, parser: parser}
}

type Scheduler struct {
	scheduler *cron.Cron
	parser    cron.Parser
	entries   map[string]EntryID
}

func (c *Scheduler) Stop() context.Context {
	return c.scheduler.Stop()
}

func (c *Scheduler) Add(cronString string, cmd func()) error {
	_, err := c.scheduler.AddFunc(cronString, cmd)
	return err
}

func (c *Scheduler) AddJob(key string, cronString string, cmd func()) error {
	id, err := c.scheduler.AddFunc(cronString, cmd)
	if err != nil {
		return err
	}
	if c.entries == nil {
		c.entries = make(map[string]EntryID)
	}
	c.entries[key] = id
	return nil
}

func (c *Scheduler) RemoveJob(key string) {
	if id, ok := c.entries[key]; ok {
		c.scheduler.Remove(id)
		delete(c.entries, key)
	}
}

func (c *Scheduler) HasJob(key string) bool {
	_, ok := c.entries[key]
	return ok
}

func (c *Scheduler) GetParser() *cron.Parser { return &c.parser }
