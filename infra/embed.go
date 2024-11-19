package infra

import "embed"

//go:embed local
//go:embed influxdb
//go:embed natsio
//go:embed promstack
//go:embed surrealdb
var LocalInfraFS embed.FS
