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

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	log.Printf("retention scheduler started (interval: %v)", s.Interval)

	for {
		select {
		case <-ticker.C:
			log.Println("retention cleanup: starting automatic cleanup")
			if err := s.Service.CleanupByRetention(ctx); err != nil {
				log.Printf("retention cleanup error: %v", err)
			} else {
				log.Println("retention cleanup: completed successfully")
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

func (s *Scheduler) Stop() {
	close(s.stopChan)
}
