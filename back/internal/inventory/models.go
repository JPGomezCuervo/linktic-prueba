package inventory

type item struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Units     int    `json:"units"`
	Price     int    `json:"price"`
	Deleted   bool   `json:"deleted"`
	CreatedAt int    `json:"createdAt"`
	UpdatedAt int    `json:"updatedAt"`
}

type auditFieldChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

type itemAuditEntry struct {
	ID             string                      `json:"id"`
	ItemID         string                      `json:"itemId"`
	Operation      string                      `json:"operation"`
	Changes        map[string]auditFieldChange `json:"changes"`
	ActorAccountID string                      `json:"actorAccountId"`
	ActorName      string                      `json:"actorName"`
	ActorEmail     string                      `json:"actorEmail"`
	CreatedAt      int                         `json:"createdAt"`
}

type itemDetails struct {
	Item    *item            `json:"item"`
	History []*itemAuditEntry `json:"history"`
}

type inventory struct {
	Items           []*item `json:"items"`
	HasNextPage     bool    `json:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage"`
	StartCursor     string  `json:"startCursor"`
	EndCursor       string  `json:"endCursor"`
}

type getInventoryInput struct {
	Items     string
	After     string
	Name      string
	UnitsMin  string
	UnitsMax  string
	PriceMin  string
	PriceMax  string
	SortBy    string
	SortOrder string
}

type inventoryQuery struct {
	Limit     int
	After     string
	Name      string
	UnitsMin  *int
	UnitsMax  *int
	PriceMin  *int
	PriceMax  *int
	SortBy    string
	SortOrder string
}

type inventoryPage struct {
	Items           []*item
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     string
	EndCursor       string
}

type createItemInput struct {
	Name  string `json:"name"`
	Units int    `json:"units"`
	Price int    `json:"price"`
}

type updateItemInput struct {
	Name  *string `json:"name"`
	Units *int    `json:"units"`
	Price *int    `json:"price"`
}

type restockItemInput struct {
	Units         int    `json:"units"`
	PaymentMethod string `json:"paymentMethod"`
}

type order struct {
	ID               string `json:"id"`
	AccountID        string `json:"accountId"`
	ItemID           string `json:"itemId"`
	ItemName         string `json:"itemName"`
	Units            int    `json:"units"`
	UnitPrice        int    `json:"unitPrice"`
	TotalPrice       int    `json:"totalPrice"`
	PaymentMethod    string `json:"paymentMethod"`
	Status           string `json:"status"`
	DeliveryAt       int    `json:"deliveryAt"`
	CompletedAt      *int   `json:"completedAt"`
	CreatedAt        int    `json:"createdAt"`
	UpdatedAt        int    `json:"updatedAt"`
	RemainingSeconds int    `json:"remainingSeconds"`
}

type orders struct {
	Orders          []*order `json:"orders"`
	HasNextPage     bool     `json:"hasNextPage"`
	HasPreviousPage bool     `json:"hasPreviousPage"`
	StartCursor     string   `json:"startCursor"`
	EndCursor       string   `json:"endCursor"`
}

type getOrdersInput struct {
	Items string
	After string
}

type ordersQuery struct {
	Limit     int
	After     string
	AccountID string
}

type ordersPage struct {
	Orders          []*order
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     string
	EndCursor       string
}

type restockResult struct {
	Order           *order `json:"order"`
	DeliverySeconds int    `json:"deliverySeconds"`
}
