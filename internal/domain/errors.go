package domain

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrValidation          = errors.New("validation error")
	ErrSAPSyncFailed       = errors.New("sap sync failed")
	ErrOrderNumberRequired = errors.New("orderNumber is required")
	ErrStatusRequired      = errors.New("status is required")
	ErrItemFieldsRequired  = errors.New("demandCode, description and deliveryDate are required")
	ErrInvalidDeliveryDate = errors.New("invalid deliveryDate format, expected YYYY-MM-DD")
	ErrSyncNotCancellable  = errors.New("sync can only be cancelled while pending")
)
