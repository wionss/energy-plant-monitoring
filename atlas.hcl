data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/atlasloader",
  ]
}

env "local" {
  src = data.external_schema.gorm.url
  dev = "postgres://${getenv("ATLAS_DATABASE_USER")}:${getenv("ATLAS_DATABASE_PASSWORD")}@${getenv("ATLAS_DATABASE_HOST")}:${getenv("ATLAS_DATABASE_PORT")}/${getenv("ATLAS_DATABASE_NAME")}?search_path=public&sslmode=disable"
  schemas = ["public", "master", "operational", "analytical"]
  migration {
    dir = "file://migrations?format=goose"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
  diff {
    skip {
      drop_schema = true
    }
  }
}
