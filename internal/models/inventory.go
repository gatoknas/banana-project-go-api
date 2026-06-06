package models

import "time"

// Inventory represents physical stock tracking for a product (Spanish: Inventario).
type Inventory struct {
	// ProductID is the reference to the product (Spanish: ID Producto).
	ProductID int64 `json:"productId" db:"product_id"`
	// CurrentStock is the real-time physical stock level (Spanish: Stock Actual).
	CurrentStock float64 `json:"currentStock" db:"current_stock"`
	// MinimumStock is the reorder threshold (Spanish: Stock Mínimo).
	MinimumStock float64 `json:"minimumStock" db:"minimum_stock"`
	// MaximumStock is the maximum storage limit (Spanish: Stock Máximo).
	MaximumStock float64 `json:"maximumStock" db:"maximum_stock"`
	// UpdatedAt is the last time the stock was adjusted (Spanish: Actualizado En).
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// Supplier represents a goods provider (Spanish: Proveedor).
type Supplier struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// TaxID is the tax identifier (e.g. NIT/Cédula) (Spanish: NIT o Cédula).
	TaxID string `json:"taxId" db:"tax_id"`
	// CompanyName is the legal name of the business (Spanish: Razón Social).
	CompanyName string `json:"companyName" db:"company_name"`
	// ContactName is the provider contact representative (Spanish: Nombre de Contacto).
	ContactName *string `json:"contactName" db:"contact_name"`
	// Phone is the contact number (Spanish: Teléfono).
	Phone *string `json:"phone" db:"phone"`
	// Email is the contact email address (Spanish: Correo Electrónico).
	Email *string `json:"email" db:"email"`
	// CreatedAt is the timestamp when the supplier was registered (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// Purchase represents a purchase order or restock receipt (Spanish: Compra).
type Purchase struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// SupplierID is the reference to the supplying company (Spanish: ID Proveedor).
	SupplierID int64 `json:"supplierId" db:"supplier_id"`
	// PurchaseDate is when the transaction occurred (Spanish: Fecha de Compra).
	PurchaseDate time.Time `json:"purchaseDate" db:"purchase_date"`
	// InvoiceNumber is the provider's physical invoice ID (Spanish: Número de Factura).
	InvoiceNumber *string `json:"invoiceNumber" db:"invoice_number"`
	// TotalAmount is the sum paid for the purchase (Spanish: Total de la Compra).
	TotalAmount float64 `json:"totalAmount" db:"total_amount"`
	// Notes is any additional comment or observation (Spanish: Observaciones).
	Notes *string `json:"notes" db:"notes"`
	// CreatedAt is the creation timestamp (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// PurchaseDetail represents a single line-item of a purchase invoice (Spanish: Detalle Compra).
type PurchaseDetail struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// PurchaseID is the parent purchase header ID (Spanish: ID Compra).
	PurchaseID int64 `json:"purchaseId" db:"purchase_id"`
	// ProductID is the bought item or ingredient product ID (Spanish: ID Producto).
	ProductID int64 `json:"productId" db:"product_id"`
	// QuantityPurchased is the quantity bought in the product's unit (Spanish: Cantidad Comprada).
	QuantityPurchased float64 `json:"quantityPurchased" db:"quantity_purchased"`
	// UnitCost is the net cost paid per unit in this purchase (Spanish: Costo Unitario).
	UnitCost float64 `json:"unitCost" db:"unit_cost"`
}
