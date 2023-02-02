// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mkungla/happy/pkg/address"
	"github.com/mkungla/happy/pkg/vars"
	"github.com/robfig/cron/v3"
	"golang.org/x/exp/slog"
)

type Service struct {
	name string

	EventListener
	TickerFuncs

	initializeAction Action
	startAction      Action
	stopAction       Action
	tickAction       ActionTick
	tockAction       ActionTock
	listeners        map[string][]ActionWithEvent

	cronsetup func(schedule CronScheduler)
}

// NewService cretes new draft service which you can compose
// before passing it to applciation or providing it from addon.
func NewService(name string, opts ...OptionArg) *Service {
	svc := &Service{
		name: name,
	}
	return svc
}

// OnInitialize is called when app is preparing runtime
// and attaching services.
func (s *Service) OnInitialize(action Action) {
	s.initializeAction = action
}

// OnStart is called when service is requested to be started.
// For instace when command is requiring this service or whenever
// service is required on runtime via sess.RequireService call.
//
// Start can be called multiple times in case of service restarts.
// If you do not want to allow service restarts you should implement
// your logic in OnStop when it's called first time and check that
// state OnStart.
func (s *Service) OnStart(action Action) {
	s.startAction = action
}

// OnStop is called when runtime request to stop the service is recieved.
func (s *Service) OnStop(action Action) {
	s.stopAction = action
}

// OnTick when set will be called every application tick when service is in running state.
func (s *Service) OnTick(action ActionTick) {
	s.tickAction = action
}

// OnTock is called after every tick.
func (s *Service) OnTock(action ActionTock) {
	s.tockAction = action
}

// OnEvent is called when a specific event is received.
func (s *Service) OnEvent(scope, key string, cb ActionWithEvent) {
	if s.listeners == nil {
		s.listeners = make(map[string][]ActionWithEvent)
	}

	lid := scope + "." + key
	s.listeners[lid] = append(s.listeners[lid], cb)
}

// OnAnyEvent called when any event is received.
func (s *Service) OnAnyEvent(cb ActionWithEvent) {
	if s.listeners == nil {
		s.listeners = make(map[string][]ActionWithEvent)
	}
	s.listeners["any"] = append(s.listeners["any"], cb)
}

// Cron scheduled cron jobs to run when the service is running.
func (s *Service) Cron(setupFunc func(schedule CronScheduler)) {
	s.cronsetup = setupFunc
}

func (s *Service) container(sess *Session, addr *address.Address) *serviceContainer {
	c := &serviceContainer{}
	c.svc = s
	c.info.addr = addr
	c.info.name = s.name
	return c
}

type ServiceLoader struct {
	loading  bool
	loaderCh chan struct{}
	errs     []error
	sess     *Session
	hostaddr *address.Address
	svcs     []*address.Address
}

func NewServiceLoader(sess *Session, svcs ...string) *ServiceLoader {
	loader := &ServiceLoader{
		sess:     sess,
		loaderCh: make(chan struct{}),
	}
	hostaddr, err := address.Parse(sess.Get("app.host.addr").String())
	if err != nil {
		loader.addErr(err)
		loader.addErr(fmt.Errorf(
			"%w: loader requires valid app.host.addr",
			ErrService,
		))
	}
	loader.hostaddr = hostaddr
	for _, addr := range svcs {
		svc, err := hostaddr.ResolveService(addr)
		if err != nil {
			loader.addErr(err)
		} else {
			loader.svcs = append(loader.svcs, svc)
		}
	}

	return loader
}

func (sl *ServiceLoader) Load() <-chan struct{} {
	if sl.loading {
		return sl.loaderCh
	}
	sl.loading = true
	if len(sl.errs) > 0 {
		sl.cancel(fmt.Errorf(
			"%w: loader initializeton failed",
			ErrService,
		))
		return sl.loaderCh
	}

	timeout := time.Duration(sl.sess.Get("app.service.loader.timeout").Int64())
	if timeout <= 0 {
		timeout = time.Duration(time.Second * 30)
		sl.sess.Log().SystemDebug(
			"service loader using default timeout",
			slog.Duration("timeout", timeout),
			slog.Int64("app.service.loader.timeout", sl.sess.Get("app.service.loader.timeout").Int64()),
		)
	}

	sl.sess.Log().SystemDebug(
		"loading services...",
		slog.String("host", sl.hostaddr.Host),
		slog.String("instance", sl.hostaddr.Instance))

	queue := make(map[string]*ServiceInfo)
	var require []string

	for _, svcaddr := range sl.svcs {
		svcaddrstr := svcaddr.String()
		info, err := sl.sess.ServiceInfo(svcaddrstr)
		if err != nil {
			sl.cancel(err)
			return sl.loaderCh
		}
		if _, ok := queue[svcaddrstr]; ok {
			sl.cancel(fmt.Errorf(
				"%w: duplicated service request %s",
				ErrService,
				svcaddrstr,
			))
			return sl.loaderCh
		}
		if info.Running() {
			sl.sess.Log().SystemDebug(
				"requested service is already running",
				slog.String("service", svcaddrstr),
			)
			continue
		}
		sl.sess.Log().SystemDebug(
			"requesting service",
			slog.String("service", svcaddrstr),
		)
		queue[svcaddrstr] = info
		require = append(require, svcaddrstr)
	}

	sl.sess.Dispatch(StartServicesEvent(require...))

	ctx, cancel := context.WithTimeout(sl.sess, timeout)

	go func() {
		defer cancel()
		ltick := time.NewTicker(time.Millisecond * 100)
		defer ltick.Stop()
		qlen := len(queue)

	loader:
		for {
			select {
			case <-ctx.Done():
				sl.sess.Log().Warn("loader context done")
				for _, status := range queue {
					if !status.Running() {
						sl.addErr(fmt.Errorf("service did not load on time %s", status.Addr().String()))
					}
				}
				sl.cancel(ctx.Err())
				return
			case <-ltick.C:
				var loaded int
				for _, status := range queue {
					if errs := status.Errs(); errs != nil {
						for _, err := range errs {
							sl.addErr(err)
						}
						sl.cancel(fmt.Errorf("%w: service loader failed to load required services %s", ErrService, status.Addr().String()))
						return
					}
					if status.Running() {
						loaded++
					}
				}
				if loaded == qlen {
					break loader
				}
			}
		}
		sl.done()
	}()

	return sl.loaderCh
}

func (sl *ServiceLoader) Err() error {
	if sl.loading {
		return fmt.Errorf("%w: service loader error checked before loader finished! did you wait for .Loaded?", ErrService)
	}
	return errors.Join(sl.errs...)
}

// cancel is used internally to cancel loading
func (sl *ServiceLoader) cancel(reason error) {
	sl.sess.Log().Warn("sevice loader canceled", slog.Any("reason", reason))
	sl.addErr(reason)
	sl.loading = false
	defer close(sl.loaderCh)
	return
}

func (sl *ServiceLoader) done() {
	sl.loading = false
	sl.sess.Log().Debug("service loader completed")
	defer close(sl.loaderCh)
}

func (sl *ServiceLoader) addErr(err error) {
	if err == nil {
		return
	}
	sl.errs = append(sl.errs, err)
}

func StartServicesEvent(svcs ...string) Event {
	var payload vars.Map
	for i, url := range svcs {
		payload.Store(fmt.Sprintf("service.%d", i), url)
	}

	return NewEvent("services", "start.services", &payload, nil)
}

func StopServicesEvent(svcs ...string) Event {
	var payload vars.Map
	for i, url := range svcs {
		payload.Store(fmt.Sprintf("service.%d", i), url)
	}

	return NewEvent("services", "stop.services", &payload, nil)
}

type ServiceInfo struct {
	mu        sync.RWMutex
	name      string
	addr      *address.Address
	running   bool
	errs      map[time.Time]error
	startedAt time.Time
	stoppedAt time.Time
}

func (s *ServiceInfo) Running() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *ServiceInfo) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

func (s *ServiceInfo) StartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startedAt
}

func (s *ServiceInfo) StoppedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stoppedAt
}

func (s *ServiceInfo) Addr() *address.Address {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

func (s *ServiceInfo) Failed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.errs) > 0
}

func (s *ServiceInfo) Errs() map[time.Time]error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errs
}

func (s *ServiceInfo) started() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
	s.startedAt = time.Now().UTC()
}

func (s *ServiceInfo) stopped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	s.stoppedAt = time.Now().UTC()
}

func (s *ServiceInfo) addErr(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.errs == nil {
		s.errs = make(map[time.Time]error)
	}
	s.errs[time.Now().UTC()] = err
}

type serviceContainer struct {
	mu     sync.Mutex
	info   ServiceInfo
	svc    *Service
	cancel context.CancelCauseFunc
	ctx    context.Context
	cron   *Cron
}

func (s *serviceContainer) initialize(sess *Session) error {
	if s.svc.initializeAction != nil {
		if err := s.svc.initializeAction(sess); err != nil {
			s.info.addErr(err)
			return err
		}
	}

	if s.svc.cronsetup != nil {
		s.cron = newCron(sess)
		s.svc.cronsetup(s.cron)
	}
	sess.Log().Debug("service initialied", slog.String("service", s.info.Addr().String()))
	return nil
}

func (s *serviceContainer) start(ectx context.Context, sess *Session) (err error) {
	if s.svc.startAction != nil {
		err = s.svc.startAction(sess)
	}
	if s.cron != nil {
		sess.Log().SystemDebug("starting cron jobs", slog.String("service", s.info.Addr().String()))
		s.cron.Start()
	}

	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancelCause(ectx) // with engine context
	s.mu.Unlock()

	if err == nil {
		s.info.started()
	} else {
		s.info.addErr(err)
	}

	payload := new(vars.Map)
	payload.Store("name", s.info.Name())
	payload.Store("addr", s.info.Addr())
	if err != nil {
		payload.Store("err", err)
	}
	payload.Store("running", s.info.Running())
	payload.Store("started.at", s.info.StartedAt())
	sess.Dispatch(NewEvent("services", "service.started", payload, nil))

	sess.Log().Debug("service started", slog.String("service", s.info.Addr().String()))
	return nil
}

func (s *serviceContainer) stop(sess *Session, e error) (err error) {
	if e != nil {
		sess.Log().Error("service error", e, slog.String("service", s.info.Addr().String()))
	}
	if s.cron != nil {
		sess.Log().SystemDebug("stopping cron scheduler, waiting jobs to finish", slog.String("service", s.info.Addr().String()))
		s.cron.Stop()
	}

	s.cancel(e)
	if s.svc.stopAction != nil {
		err = s.svc.stopAction(sess)
	}

	if e != nil {
		err = errors.Join(err, e)
	}

	s.info.stopped()

	payload := new(vars.Map)
	payload.Store("name", s.info.Name())
	payload.Store("addr", s.info.Addr())
	if err != nil {
		payload.Store("err", err)
	}
	payload.Store("running", s.info.Running())
	payload.Store("stopped.at", s.info.StoppedAt())
	sess.Dispatch(NewEvent("services", "service.stopped", payload, nil))

	sess.Log().Debug("service stopped", slog.String("service", s.info.Addr().String()))
	return err
}

func (s *serviceContainer) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	done := s.ctx.Done()
	return done
}

func (s *serviceContainer) tick(sess *Session, ts time.Time, delta time.Duration) error {
	if s.svc.tickAction == nil {
		return nil
	}
	return s.svc.tickAction(sess, ts, delta)
}

func (s *serviceContainer) tock(sess *Session, delta time.Duration, tps int) error {
	if s.svc.tockAction == nil {
		return nil
	}
	return s.svc.tockAction(sess, delta, tps)
}

func (s *serviceContainer) handleEvent(sess *Session, ev Event) {
	if s.svc.listeners == nil {
		return
	}
	lid := ev.Scope() + "." + ev.Key()
	for sk, listeners := range s.svc.listeners {
		for _, listener := range listeners {
			if sk == "any" || sk == lid {
				if err := listener(sess, ev); err != nil {
					s.info.addErr(err)
					sess.Log().Error("event handler error", err, slog.String("service", s.info.Addr().String()))
				}
			}
		}
	}
}

type CronScheduler interface {
	Job(expr string, cb Action)
}

type Cron struct {
	sess   *Session
	lib    *cron.Cron
	jobIDs []cron.EntryID
}

func newCron(sess *Session) *Cron {
	c := &Cron{}
	c.sess = sess
	c.lib = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	return c
}

func (cs *Cron) Job(expr string, cb Action) {
	id, err := cs.lib.AddFunc(expr, func() {
		if err := cb(cs.sess); err != nil {
			cs.sess.Log().Error("job failed", err)
		}
	})
	cs.jobIDs = append(cs.jobIDs, id)
	if err != nil {
		cs.sess.Log().Error("failed to add job", err, slog.Int("id", int(id)))
		return
	}
}

func (cs *Cron) Start() error {
	if cs.sess.Get("app.cron.on.service.start").Bool() {
		for _, id := range cs.jobIDs {
			cs.sess.Log().SystemDebug(
				"executing cron first time",
				slog.Int("id", int(id)),
			)
			job := cs.lib.Entry(id)
			if job.Job != nil {
				go job.Job.Run()
			}
		}
	}
	cs.lib.Start()
	return nil
}

func (cs *Cron) Stop() error {
	ctx := cs.lib.Stop()
	<-ctx.Done()
	return nil
}
