package handlers

import "org.banana.project/api/internal/service"

// MessageResponse is the standard body returned by create, update, and delete handlers.
type MessageResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	ID      int64  `json:"id,omitempty"`
}

// SaleResponse is the body returned when a sale is created.
type SaleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	SaleID  int64  `json:"saleId"`
}

// SyncResponse is the body returned when email receipts are synced.
type SyncResponse struct {
	Status  string             `json:"status"`
	Message string             `json:"message"`
	Result  service.SyncResult `json:"result"`
}
