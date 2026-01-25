package repositories

import (
	"log"
	"sync"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"gorm.io/gorm"
)

const (
	asyncBufferSize = 1000
)

// eventPair contiene ambos eventos para escritura async
type eventPair struct {
	operational *entities.EventOperational
	analytical  *entities.EventAnalytical
}

// DualEventWriter implementa escritura dual a operational y analytical
//
// PROPÓSITO:
// Garantiza que cada evento se escriba tanto en la tabla operacional (datos calientes)
// como en la tabla analítica (TimescaleDB hypertable) de forma consistente.
//
// MODOS DE OPERACIÓN:
// - SaveEvent: Escritura síncrona con transacción atómica
// - SaveEventAsync: Escritura asíncrona con buffer y workers en background
type DualEventWriter struct {
	db             *gorm.DB
	opRepo         *EventOperationalRepository
	anRepo         *EventAnalyticalRepository
	asyncChannel   chan eventPair
	workerCount    int
	wg             sync.WaitGroup
	stopOnce       sync.Once
	stopChan       chan struct{}
}

var _ output.DualEventWriterInterface = &DualEventWriter{}

// NewDualEventWriter crea una nueva instancia del escritor dual
// workerCount: número de workers para escritura async (recomendado: 4)
func NewDualEventWriter(
	db *gorm.DB,
	opRepo *EventOperationalRepository,
	anRepo *EventAnalyticalRepository,
	workerCount int,
) *DualEventWriter {
	if workerCount <= 0 {
		workerCount = 4
	}

	writer := &DualEventWriter{
		db:           db,
		opRepo:       opRepo,
		anRepo:       anRepo,
		asyncChannel: make(chan eventPair, asyncBufferSize),
		workerCount:  workerCount,
		stopChan:     make(chan struct{}),
	}

	// Iniciar workers
	for i := 0; i < workerCount; i++ {
		writer.wg.Add(1)
		go writer.worker(i)
	}

	log.Printf("DualEventWriter initialized with %d workers and buffer size %d", workerCount, asyncBufferSize)
	return writer
}

// SaveEvent escribe en ambas tablas de forma síncrona con transacción
func (w *DualEventWriter) SaveEvent(op *entities.EventOperational, an *entities.EventAnalytical) error {
	return w.db.Transaction(func(tx *gorm.DB) error {
		// Crear repositorios temporales con la transacción
		opRepoTx := &EventOperationalRepository{db: tx}
		anRepoTx := &EventAnalyticalRepository{db: tx}

		// Escribir en operational
		if _, err := opRepoTx.Create(op); err != nil {
			log.Printf("ERROR: Failed to write to operational: %v", err)
			return err
		}

		// Escribir en analytical
		if _, err := anRepoTx.Create(an); err != nil {
			log.Printf("ERROR: Failed to write to analytical: %v", err)
			return err
		}

		log.Printf("✓ Event saved to both operational and analytical: ID=%s", op.ID)
		return nil
	})
}

// SaveEventAsync encola el evento para escritura asíncrona
// Si el buffer está lleno, hace fallback a escritura síncrona
func (w *DualEventWriter) SaveEventAsync(op *entities.EventOperational, an *entities.EventAnalytical) error {
	select {
	case w.asyncChannel <- eventPair{operational: op, analytical: an}:
		// Evento encolado exitosamente
		return nil
	default:
		// Buffer lleno, fallback a escritura síncrona
		log.Printf("WARNING: Async buffer full, falling back to sync write for event ID=%s", op.ID)
		return w.SaveEvent(op, an)
	}
}

// worker procesa eventos del canal async
func (w *DualEventWriter) worker(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopChan:
			log.Printf("Worker %d stopping", id)
			return
		case pair, ok := <-w.asyncChannel:
			if !ok {
				log.Printf("Worker %d: channel closed", id)
				return
			}
			if err := w.SaveEvent(pair.operational, pair.analytical); err != nil {
				log.Printf("Worker %d: ERROR saving event ID=%s: %v", id, pair.operational.ID, err)
			}
		}
	}
}

// Stop detiene los workers y espera a que terminen
func (w *DualEventWriter) Stop() {
	w.stopOnce.Do(func() {
		log.Println("Stopping DualEventWriter...")
		close(w.stopChan)
		close(w.asyncChannel)
		w.wg.Wait()
		log.Println("DualEventWriter stopped")
	})
}
