package models

import "time"

// Sale represents a POS sales transaction ticket (Spanish: Venta).
type Sale struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// UserID is the reference ID of the cashier who sold it (Spanish: ID Usuario).
	UserID int64 `json:"userId" db:"user_id"`
	// UserName is the name of the cashier / user who registered the sale (Spanish: Nombre Usuario).
	UserName *string `json:"userName,omitempty" db:"user_name"`
	// SaleDate is when the checkout occurred (Spanish: Fecha de Venta).
	SaleDate time.Time `json:"saleDate" db:"sale_date"`
	// TotalAmount is the final total net amount received (Spanish: Total de la Venta).
	TotalAmount float64 `json:"totalAmount" db:"total_amount"`
	// PaymentMethod is the checkout method (e.g. Cash, Card) (Spanish: Método de Pago).
	PaymentMethod string `json:"paymentMethod" db:"payment_method"`
	// ItemsCount is the total number of distinct line items in the sale (Spanish: Cantidad de Ítems).
	ItemsCount int `json:"itemsCount,omitempty" db:"items_count"`
	// CreatedAt is the creation timestamp (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// Details is the list of line items in this sale (Spanish: Detalles de Venta).
	Details []SaleDetail `json:"details,omitempty"`
}

// SaleDetail represents a single product line-item in a sales ticket (Spanish: Detalle Venta).
type SaleDetail struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// SaleID is the parent sale header ID (Spanish: ID Venta).
	SaleID int64 `json:"saleId" db:"sale_id"`
	// ProductID is the principal sold product ID (Spanish: ID Producto).
	ProductID int64 `json:"productId" db:"product_id"`
	// ProductName is the name of the sold product (Spanish: Nombre Producto).
	ProductName *string `json:"productName,omitempty" db:"product_name"`
	// Quantity is the amount sold, supports decimals for weighed items (Spanish: Cantidad).
	Quantity float64 `json:"quantity" db:"quantity"`
	// HistoricalUnitPrice is the unit price at the time of sale (Spanish: Precio Unitario Histórico).
	HistoricalUnitPrice float64 `json:"historicalUnitPrice" db:"historical_unit_price"`
	// Subtotal is quantity * HistoricalUnitPrice (Spanish: Subtotal).
	Subtotal float64 `json:"subtotal" db:"subtotal"`
}

// AdditionDetail represents extra toppings or modifications attached to a specific line-item (Spanish: Detalle Adición).
type AdditionDetail struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// SaleDetailID is the parent sale line-item ID (Spanish: ID Detalle Venta).
	SaleDetailID int64 `json:"saleDetailId" db:"sale_detail_id"`
	// AdditionProductID is the reference to the topping product ID (Spanish: ID Producto Adición).
	AdditionProductID int64 `json:"additionProductId" db:"addition_product_id"`
	// Quantity is the count of portions added (Spanish: Cantidad).
	Quantity float64 `json:"quantity" db:"quantity"`
	// HistoricalAdditionPrice is the topping surcharge price at the time of sale (Spanish: Precio de Adición Histórico).
	HistoricalAdditionPrice float64 `json:"historicalAdditionPrice" db:"historical_addition_price"`
}

// SaleFilter defines query parameters for listing sales.
type SaleFilter struct {
	Page          int        `json:"page"`
	PageSize      int        `json:"pageSize"`
	FromDate      *time.Time `json:"fromDate"`
	ToDate        *time.Time `json:"toDate"`
	PaymentMethod string     `json:"paymentMethod"`
	UserID        *int64     `json:"userId"`
}

// SalePagination represents pagination metadata for a sales list.
type SalePagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

// SaleSummaryStats represents aggregated metrics for the filtered set of sales.
type SaleSummaryStats struct {
	TotalAmount float64 `json:"totalAmount"`
	TotalCount  int64   `json:"totalCount"`
}

// PaginatedSalesResponse wraps a paginated list of sales with metadata and summary.
type PaginatedSalesResponse struct {
	Data       []Sale           `json:"data"`
	Pagination SalePagination   `json:"pagination"`
	Summary    SaleSummaryStats `json:"summary"`
}
