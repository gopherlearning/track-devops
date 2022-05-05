package migrations

import "embed"

//go:embed *.sql
var Migrations embed.FS

// var Migrations = map[string]string{
// 	"s_001_create_metrics": s_001_create_metrics,
// }

// var (
// 	// go:embed 001_create_metrics.sql
// 	s_001_create_metrics string
// )
