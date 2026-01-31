package main

import (
	"io"
	"log/slog"
	"os"

	"monitoring-energy-service/internal/infrastructure/adapters/repositories"

	"ariga.io/atlas-provider-gorm/gormschema"
)

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&repositories.ExampleModel{},
		&repositories.EnergyPlantsModel{},
		&repositories.EventModel{},
		&repositories.EventOperationalModel{},
		&repositories.EventAnalyticalModel{},
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
