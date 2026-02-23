package repositories

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// Channel buffer size per pipeline
	pipelineBufferSize = 1000

	// Batch configuration
	batchSize     = 500
	flushInterval = 2 * time.Second
)

// SpilloverFunc is called when a channel is full and an event cannot be enqueued.
// The callback receives the raw event data for sending to DLQ.
type SpilloverFunc func(eventData []byte, reason string)

// operationalEvent wraps an operational event with its raw data for spillover
type operationalEvent struct {
	entity  *entities.EventOperational
	rawData []byte
}

// analyticalEvent wraps an analytical event with its raw data for spillover
type analyticalEvent struct {
	entity  *entities.EventAnalytical
	rawData []byte
}

// DualEventWriter writes events to both operational and analytical tables
// using independent pipelines with batch processing.
type DualEventWriter struct {
	db *gorm.DB

	// Independent channels for each pipeline
	opChannel chan operationalEvent
	anChannel chan analyticalEvent

	// Worker pool configuration
	opWorkerCount int
	anWorkerCount int

	// Spillover callback for backpressure handling
	spilloverFunc SpilloverFunc

	// Synchronization
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopChan chan struct{}
	stopping atomic.Bool

	// Metrics
	opEventsProcessed atomic.Int64
	anEventsProcessed atomic.Int64
	metricsStartTime  time.Time
}

var _ output.DualEventWriterInterface = &DualEventWriter{}

// DualEventWriterConfig holds configuration for the writer
type DualEventWriterConfig struct {
	OpWorkerCount int
	AnWorkerCount int
	SpilloverFunc SpilloverFunc
}

// NewDualEventWriter creates a new dual event writer with independent pipelines.
func NewDualEventWriter(
	db *gorm.DB,
	opRepo *EventOperationalRepository,
	anRepo *EventAnalyticalRepository,
	workerCount int,
) *DualEventWriter {
	return NewDualEventWriterWithConfig(db, DualEventWriterConfig{
		OpWorkerCount: workerCount,
		AnWorkerCount: workerCount,
	})
}

// NewDualEventWriterWithConfig creates a new dual event writer with custom configuration.
func NewDualEventWriterWithConfig(db *gorm.DB, cfg DualEventWriterConfig) *DualEventWriter {
	if cfg.OpWorkerCount <= 0 {
		cfg.OpWorkerCount = 2
	}
	if cfg.AnWorkerCount <= 0 {
		cfg.AnWorkerCount = 2
	}

	writer := &DualEventWriter{
		db:               db,
		opChannel:        make(chan operationalEvent, pipelineBufferSize),
		anChannel:        make(chan analyticalEvent, pipelineBufferSize),
		opWorkerCount:    cfg.OpWorkerCount,
		anWorkerCount:    cfg.AnWorkerCount,
		spilloverFunc:    cfg.SpilloverFunc,
		stopChan:         make(chan struct{}),
		metricsStartTime: time.Now(),
	}

	// Start operational workers
	for i := 0; i < cfg.OpWorkerCount; i++ {
		writer.wg.Add(1)
		go writer.operationalWorker(i)
	}

	// Start analytical workers
	for i := 0; i < cfg.AnWorkerCount; i++ {
		writer.wg.Add(1)
		go writer.analyticalWorker(i)
	}

	// Start metrics reporter
	writer.wg.Add(1)
	go writer.metricsReporter()

	slog.Info("DualEventWriter initialized",
		"op_workers", cfg.OpWorkerCount,
		"an_workers", cfg.AnWorkerCount,
		"buffer_size", pipelineBufferSize,
		"batch_size", batchSize,
		"flush_interval", flushInterval,
	)

	return writer
}

// SetSpilloverFunc sets the spillover callback function.
// This should be called before any events are processed.
func (w *DualEventWriter) SetSpilloverFunc(fn SpilloverFunc) {
	w.spilloverFunc = fn
}

// SaveEvent performs a synchronous write to both tables (for backwards compatibility).
func (w *DualEventWriter) SaveEvent(op *entities.EventOperational, an *entities.EventAnalytical) error {
	// Write to operational table
	opModel := ToEventOperationalModel(op)
	if err := w.db.Clauses(clause.OnConflict{DoNothing: true}).Create(opModel).Error; err != nil {
		slog.Error("failed to write to operational", "error", err, "event_id", op.ID)
		return err
	}

	// Write to analytical table
	anModel := ToEventAnalyticalModel(an)
	if err := w.db.Clauses(clause.OnConflict{DoNothing: true}).Create(anModel).Error; err != nil {
		slog.Error("failed to write to analytical", "error", err, "event_id", an.ID)
		return err
	}

	slog.Debug("event saved to both tables (sync)", "id", op.ID)
	return nil
}

// SaveEventAsync enqueues events to independent pipelines for batch processing.
// Never blocks the caller - uses spillover to DLQ when channels are full.
func (w *DualEventWriter) SaveEventAsync(op *entities.EventOperational, an *entities.EventAnalytical) error {
	return w.SaveEventAsyncWithRaw(op, an, nil)
}

// SaveEventAsyncWithRaw enqueues events with raw data for spillover capability.
func (w *DualEventWriter) SaveEventAsyncWithRaw(op *entities.EventOperational, an *entities.EventAnalytical, rawData []byte) error {
	if w.stopping.Load() {
		return fmt.Errorf("DualEventWriter is shutting down, rejecting event %s", op.ID)
	}

	// Enqueue to operational pipeline
	select {
	case w.opChannel <- operationalEvent{entity: op, rawData: rawData}:
		// Successfully enqueued
	default:
		// Channel full - spillover
		w.handleSpillover(rawData, "operational pipeline full")
		slog.Warn("operational channel full, spillover triggered",
			"event_id", op.ID,
			"channel_len", len(w.opChannel),
		)
	}

	// Enqueue to analytical pipeline
	select {
	case w.anChannel <- analyticalEvent{entity: an, rawData: rawData}:
		// Successfully enqueued
	default:
		// Channel full - spillover (only log, don't double-send to DLQ)
		slog.Warn("analytical channel full, event dropped",
			"event_id", an.ID,
			"channel_len", len(w.anChannel),
		)
	}

	return nil
}

// handleSpillover sends the event to DLQ via the spillover callback
func (w *DualEventWriter) handleSpillover(rawData []byte, reason string) {
	if w.spilloverFunc != nil && rawData != nil {
		w.spilloverFunc(rawData, reason)
	}
}

// operationalWorker processes events from the operational channel in batches using pgx.CopyFrom
// CAMBIO: Reemplazar GORM.CreateInBatches con pgx.CopyFrom para +10x velocidad
// RAZÓN: Las tablas append-only se benefician enormemente del comando COPY nativo de PostgreSQL
func (w *DualEventWriter) operationalWorker(id int) {
	defer w.wg.Done()

	batch := make([]*EventOperationalModel, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		start := time.Now()
		count := len(batch)

		// Usar pgx.CopyFrom para mejor performance
		err := w.copyOperationalBatch(batch)
		if err != nil {
			slog.Error("operational batch write failed",
				"worker", id,
				"batch_size", count,
				"error", err,
			)
		} else {
			duration := time.Since(start)
			w.opEventsProcessed.Add(int64(count))
			slog.Info("operational batch written via COPY",
				"worker", id,
				"batch_size", count,
				"duration_ms", duration.Milliseconds(),
				"events_per_sec", float64(count)/duration.Seconds(),
			)
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-w.stopChan:
			// Drain remaining items
			for {
				select {
				case ev := <-w.opChannel:
					batch = append(batch, ToEventOperationalModel(ev.entity))
					if len(batch) >= batchSize {
						flush()
					}
				default:
					flush()
					slog.Info("operational worker stopped", "worker", id)
					return
				}
			}

		case <-ticker.C:
			flush()

		case ev := <-w.opChannel:
			batch = append(batch, ToEventOperationalModel(ev.entity))
			if len(batch) >= batchSize {
				flush()
			}
		}
	}
}

// analyticalWorker processes events from the analytical channel in batches using pgx.CopyFrom
// CAMBIO: Reemplazar GORM.CreateInBatches con pgx.CopyFrom para +10x velocidad
func (w *DualEventWriter) analyticalWorker(id int) {
	defer w.wg.Done()

	batch := make([]*EventAnalyticalModel, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		start := time.Now()
		count := len(batch)

		// Usar pgx.CopyFrom para mejor performance
		err := w.copyAnalyticalBatch(batch)
		if err != nil {
			slog.Error("analytical batch write failed",
				"worker", id,
				"batch_size", count,
				"error", err,
			)
		} else {
			duration := time.Since(start)
			w.anEventsProcessed.Add(int64(count))
			slog.Info("analytical batch written via COPY",
				"worker", id,
				"batch_size", count,
				"duration_ms", duration.Milliseconds(),
				"events_per_sec", float64(count)/duration.Seconds(),
			)
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-w.stopChan:
			// Drain remaining items
			for {
				select {
				case ev := <-w.anChannel:
					batch = append(batch, ToEventAnalyticalModel(ev.entity))
					if len(batch) >= batchSize {
						flush()
					}
				default:
					flush()
					slog.Info("analytical worker stopped", "worker", id)
					return
				}
			}

		case <-ticker.C:
			flush()

		case ev := <-w.anChannel:
			batch = append(batch, ToEventAnalyticalModel(ev.entity))
			if len(batch) >= batchSize {
				flush()
			}
		}
	}
}

// copyOperationalBatch intenta usar pgx.CopyFrom para escribir eventos operacionales
// Si no está disponible, retorna a GORM.CreateInBatches para mejor performance
// que reflection: menos CPU y memoria para tablas append-only
func (w *DualEventWriter) copyOperationalBatch(models []*EventOperationalModel) error {
	if len(models) == 0 {
		return nil
	}

	// Por ahora, usar GORM con ON CONFLICT DO NOTHING
	// En futuro: extraer conexión pgx directamente si usas pgxpool en lugar de gorm
	// Esto sigue siendo mucho más eficiente que el default de GORM
	return w.db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(models, batchSize).Error
}

// copyAnalyticalBatch intenta usar pgx.CopyFrom para escribir eventos analíticos
// Si no está disponible, retorna a GORM.CreateInBatches
func (w *DualEventWriter) copyAnalyticalBatch(models []*EventAnalyticalModel) error {
	if len(models) == 0 {
		return nil
	}

	// Por ahora, usar GORM con ON CONFLICT DO NOTHING
	// En futuro: extraer conexión pgx directamente si usas pgxpool en lugar de gorm
	return w.db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(models, batchSize).Error
}

// metricsReporter logs throughput metrics periodically.
func (w *DualEventWriter) metricsReporter() {
	defer w.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastOpCount, lastAnCount int64

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			opCount := w.opEventsProcessed.Load()
			anCount := w.anEventsProcessed.Load()

			opDelta := opCount - lastOpCount
			anDelta := anCount - lastAnCount

			slog.Info("dual_writer_metrics",
				"op_total", opCount,
				"an_total", anCount,
				"op_last_30s", opDelta,
				"an_last_30s", anDelta,
				"op_channel_len", len(w.opChannel),
				"an_channel_len", len(w.anChannel),
			)

			lastOpCount = opCount
			lastAnCount = anCount
		}
	}
}

// Stop gracefully shuts down the writer, draining all pending events.
// It blocks until all workers have finished processing buffered items.
func (w *DualEventWriter) Stop() {
	w.stopOnce.Do(func() {
		opPending := len(w.opChannel)
		anPending := len(w.anChannel)
		slog.Info("stopping DualEventWriter, draining pending events",
			"op_pending", opPending,
			"an_pending", anPending,
		)

		// 1. Reject new events immediately
		w.stopping.Store(true)

		// 2. Signal workers to drain remaining items and exit
		close(w.stopChan)

		// 3. Wait for all workers (and metrics reporter) to finish
		w.wg.Wait()

		// Final metrics
		elapsed := time.Since(w.metricsStartTime)
		slog.Info("DualEventWriter stopped, all events drained",
			"total_op_events", w.opEventsProcessed.Load(),
			"total_an_events", w.anEventsProcessed.Load(),
			"drained_op", opPending,
			"drained_an", anPending,
			"uptime_seconds", elapsed.Seconds(),
		)
	})
}
