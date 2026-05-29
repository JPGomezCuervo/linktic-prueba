package telemetry

const (
	// Info Events
	OpServerStarted = "server_started"

	// Error Events
	ErrDBQuery = "db_query_failed"

	ErrInternal    = "internal_error"
	ErrJSONFailure = "json_operation_failed"
)

const (
	KeyErr         = "err"
	KeyInternalErr = "internal_err"

	KeyTable = "table"
	KeyQuery = "query"
)
