package models

import "time"

// Sale represents a POS sales transaction ticket (Spanish: Venta).
type Sale struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// UserID is the reference ID of the cashier who sold it (Spanish: ID Usuario).
	UserID int64 `json:"userId" db:"user_id"`
	// SaleDate is when the checkout occurred (Spanish: Fecha de Venta).
	SaleDate time.Time `json:"saleDate" db:"sale_date"`
	// TotalAmount is the final total net amount received (Spanish: Total de la Venta).
	TotalAmount float64 `json:"totalAmount" db:"total_amount"`
	// PaymentMethod is the checkout method (e.g. Cash, Card) (Spanish: Método de Pago).
	PaymentMethod string `json:"paymentMethod" db:"payment_method"`
	// CreatedAt is the creation timestamp (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// SaleDetail represents a single product line-item in a sales ticket (Spanish: Detalle Venta).
type SaleDetail struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// SaleID is the parent sale header ID (Spanish: ID Venta).
	SaleID int64 `json:"saleId" db:"sale_id"`
	// ProductID is the principal sold product ID (Spanish: ID Producto).
	ProductID int64 `json:"productId" db:"product_id"`
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
