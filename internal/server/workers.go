package server

import (
	"context"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/services"
)

type worker struct {
	queue    queue.Queue
	service  services.PushService
	store    store.Store // Added store
	ctx      context.Context
	cancel   context.CancelFunc
	finished chan (bool)
}

func newWorker(pp services.PushService, queue queue.Queue, st store.Store) (w *worker, err error) { // Added store
	w = &worker{
		queue:    queue,
		service:  pp,
		store:    st, // Added store
		finished: make(chan bool),
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	return
}

func (w *worker) push(msg []byte) (err error) {
	err = w.queue.Queue(msg)
	return
}

func (w *worker) serve(fc services.FeedbackCollector) {
	// How the store is passed to the service's Serve method needs consideration.
	// Option 1: Modify PushService.Serve interface to accept store.Store
	// Option 2: The service implementation accesses it via a global or context (less ideal)
	// Option 3: The FeedbackCollector could be augmented or replaced by a context carrying the store.
	// For now, let's assume PushService.Serve will be updated.
	// If PushService.Serve is not updated, then the store needs to be accessed by ServeClient differently.
	// Let's assume we will modify PushService.Serve:
	// w.service.Serve(w.ctx, w.queue, w.store, fc)
	// For now, keeping original call until PushService is modified.
	// The service will need to be aware of the new QueuedNotificationMessage format.

	// If PushService.Serve is modified to take store:
	// err := w.service.Serve(w.ctx, w.queue, w.store, fc)
	// if err != nil {
	//  log.Printf("Service %s serve error: %v", w.service.ID(), err)
	// }
	// For now, we will modify the service implementations (e.g. slack.go, email.go)
	// to handle the new message type and use the store which will be passed to their Serve methods.
	// This requires changing the services.PushService interface first.

	// Let's proceed assuming services.PushService.Serve will be updated.
	// This change will be done in the next step when modifying services.
	err := w.service.Serve(w.ctx, w.queue, w.store, fc) // fc might become redundant or part of a richer context
	if err != nil {
		log.Printf("Error from service %s: %v", w.service.ID(), err)
	}
	w.finished <- true
}

func (w *worker) shutdown() (err error) {
	if err = w.queue.Shutdown(); err != nil {
		return
	}
	w.cancel()
	<-w.finished
	return
}
