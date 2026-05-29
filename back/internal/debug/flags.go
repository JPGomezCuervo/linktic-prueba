package debug

const (
	// General
	DebugFlag = true

	// Middleware Package
	DebugMiddlewarePackageFlag = true

	// Inventory Package
	DebugInventoryPackageFlag = true
	DebugInventoryHandlerFlag = true
	DebugInventoryServiceFlag = true
	DebugInventoryStoreFlag   = true

	// Auth Package
	DebugAuthPackageFlag = true
	DebugAuthHandlerFlag = true
	DebugAuthServiceFlag = true
	DebugAuthStoreFlag   = true

	// Composed flags (layer || package)
	DebugInventoryHandler = DebugInventoryHandlerFlag || DebugInventoryPackageFlag
	DebugInventoryService = DebugInventoryServiceFlag || DebugInventoryPackageFlag
	DebugInventoryStore   = DebugInventoryStoreFlag || DebugInventoryPackageFlag
	DebugAuthHandler      = DebugAuthHandlerFlag || DebugAuthPackageFlag
	DebugAuthService      = DebugAuthServiceFlag || DebugAuthPackageFlag
	DebugAuthStore        = DebugAuthStoreFlag || DebugAuthPackageFlag
)
