package infra

import "embed"

//go:embed local
//go:embed local_infra
//go:embed influxdb
//go:embed natsio
//go:embed promstack
//go:embed surrealdb
var LocalInfraFS embed.FS
