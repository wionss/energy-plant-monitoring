package main

import (
	"io"
	"log/slog"
	"os"

	"monitoring-energy-service/internal/domain/entities"

	"ariga.io/atlas-provider-gorm/gormschema"
)

func main() {
	// Only include entities WITHOUT cross-schema dependencies
	// Entities with FK to other schemas cause GORM to include those tables too
	//
	// Atlas managed:
	// - ExampleEntity (public.examples) - no FK dependencies
	//
	// Managed via manual migrations (goose):
	// - EnergyPlants (master.energy_plants)
	// - EventEntity (public.events) - has FK to master.energy_plants
	// - EventOperational (operational.events_std)
	// - EventAnalytical (analytical.events_ts)
	stmts, err := gormschema.New("postgres").Load(
		&entities.ExampleEntity{}, // public.examples - Atlas managed
	)
	if err != nil {
		slog.Error("Failed to load gorm schema", "err", err.Error())
		os.Exit(1)
	}

	_, err = io.WriteString(os.Stdout, stmts)
	if err != nil {
		slog.Error("Failed to write gorm schema", "err", err.Error())
		os.Exit(1)
	}
}
