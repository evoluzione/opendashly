package retention

import (
	"context"
	"log"
	"time"
)

type Scheduler struct {
	Service  *CleanupService
	Interval time.Duration
	stopChan chan struct{}
}

func NewScheduler(service *CleanupService, intervalMinutes int) *Scheduler {
	return &Scheduler{
		Service:  service,
		Interval: time.Duration(intervalMinutes) * time.Minute,
		stopChan: make(chan struct{}),
	}
}

const diskCheckInterval = 5 * time.Minute

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	diskTicker := time.NewTicker(diskCheckInterval)
	defer diskTicker.Stop()
	// Run once shortly after boot: frequent (autoheal) restarts used to reset
	// the ticker before it ever fired, so retention never ran at all.
	firstRun := time.After(time.Minute)

	log.Printf("retention scheduler started (interval: %v, disk check: %v)", s.Interval, diskCheckInterval)

	for {
		select {
		case <-firstRun:
			s.runRetention(ctx)
		case <-ticker.C:
			s.runRetention(ctx)
		case <-diskTicker.C:
			if err := s.Service.PurgeForDisk(ctx); err != nil {
				log.Printf("retention disk purge error: %v", err)
			}
		case <-s.stopChan:
			log.Println("retention scheduler stopped")
			return
		case <-ctx.Done():
			log.Println("retention scheduler context cancelled")
			return
		}
	}
}

func (s *Scheduler) runRetention(ctx context.Context) {
	log.Println("retention cleanup: starting automatic cleanup")
	if err := s.Service.CleanupByRetention(ctx); err != nil {
		log.Printf("retention cleanup error: %v", err)
	} else {
		log.Println("retention cleanup: completed successfully")
	}
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
}
