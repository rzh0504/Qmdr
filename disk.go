package main

import (
	"log"
	"runtime/debug"
)

const diskQueueSize = 128

// startDiskWorker runs persistence jobs off the UI thread. The main thread
// only enqueues snapshots, and one worker writes them in order, so a slow
// disk never blocks a frame. Store operations reach the history store only
// through this worker (except the store's own setup), and the store's lock
// keeps even a late close safe.
func (a *app) startDiskWorker() {
	a.diskJobs = make(chan func(), diskQueueSize)
	go func() {
		for job := range a.diskJobs {
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						log.Printf("disk job panic: %v\n%s", recovered, debug.Stack())
					}
				}()
				job()
			}()
		}
	}()
}

// dispatchDisk enqueues a job for the disk worker. Before the worker
// exists (tests, early startup) the job runs synchronously, so ordering
// stays deterministic. When the buffer is full the job still runs off the
// UI thread, unordered with the queued ones; every job here is idempotent
// or last-write-wins, so that is safe.
func (a *app) dispatchDisk(job func()) {
	if a.diskJobs == nil {
		job()
		return
	}
	select {
	case a.diskJobs <- job:
	default:
		log.Printf("disk queue full, running job out of order")
		go job()
	}
}