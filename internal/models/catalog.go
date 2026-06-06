package models

import "time"

// UnitOfMeasure represents a unit of measurement (Spanish: Unidad de Medida).
type UnitOfMeasure struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// Name is the name of the unit (Spanish: Nombre).
	Name string `json:"name" db:"name"`
	// Abbreviation is the short code representing the unit (Spanish: Abreviatura).
	Abbreviation string `json:"abbreviation" db:"abbreviation"`
	// CreatedAt is the timestamp when the unit was created (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// Category represents a product category (Spanish: Categoría).
type Category struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// Name is the category name (Spanish: Nombre).
	Name string `json:"name" db:"name"`
	// CreatedAt is the timestamp when the category was created (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// Product represents a catalog item or ingredient (Spanish: Producto).
type Product struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// Name is the commercial name (Spanish: Nombre).
	Name string `json:"name" db:"name"`
	// Description is the optional product description (Spanish: Descripción).
	Description *string `json:"description" db:"description"`
	// CategoryID is the reference ID of the category (Spanish: ID Categoría).
	CategoryID int64 `json:"categoryId" db:"category_id"`
	// UnitOfMeasureID is the reference ID of the unit of measure (Spanish: ID Unidad de Medida).
	UnitOfMeasureID int64 `json:"unitOfMeasureId" db:"unit_of_measure_id"`
	// SellPrice is the price for customers, can be 0 for pure ingredients (Spanish: Precio de Venta).
	SellPrice float64 `json:"sellPrice" db:"sell_price"`
	// AverageCost is the dynamic weighted cost from purchases (Spanish: Costo Promedio).
	AverageCost float64 `json:"averageCost" db:"average_cost"`
	// IsForSale defines if it can be sold in the POS (Spanish: Es Para Venta).
	IsForSale bool `json:"isForSale" db:"is_for_sale"`
	// RequiresRecipe defines if it deducts child recipe items upon sale (Spanish: Requiere Receta).
	RequiresRecipe bool `json:"requiresRecipe" db:"requires_recipe"`
	// CreatedAt is the creation timestamp (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// UpdatedAt is the last modification timestamp (Spanish: Actualizado En).
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// ProductRecipe represents the Bill of Materials linking a parent product to its ingredients (Spanish: Receta Producto).
type ProductRecipe struct {
	// ParentProductID is the composite product ID (Spanish: ID Producto Padre).
	ParentProductID int64 `json:"parentProductId" db:"parent_product_id"`
	// ChildProductID is the ingredient product ID (Spanish: ID Producto Hijo).
	ChildProductID int64 `json:"childProductId" db:"child_product_id"`
	// Quantity is the amount used in the recipe in child's unit (Spanish: Cantidad).
	Quantity float64 `json:"quantity" db:"quantity"`
}

// AllowedAddition maps valid extra customizations/toppings for a product (Spanish: Adición Permitida).
type AllowedAddition struct {
	// MainProductID is the base product ID (Spanish: ID Producto Principal).
	MainProductID int64 `json:"mainProductId" db:"main_product_id"`
	// AdditionProductID is the ingredient/topping product ID (Spanish: ID Producto Adición).
	AdditionProductID int64 `json:"additionProductId" db:"addition_product_id"`
	// AdditionalPrice is the surcharge for adding this topping (Spanish: Precio Adicional).
	AdditionalPrice float64 `json:"additionalPrice" db:"additional_price"`
}
