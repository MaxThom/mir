package infra

import "embed"

//go:embed local_mir_support
//go:embed local_support
//go:embed influxdb
//go:embed natsio
//go:embed promstack
//go:embed surrealdb
//go:embed mir
var LocalInfraFS embed.FS
