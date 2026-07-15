package domain

import "errors"

var (
	ErrNotFound                 = errors.New("resource not found")
	ErrValidation               = errors.New("validation error")
	ErrSAPSyncFailed            = errors.New("sap sync failed")
	ErrOrderNumberRequired      = errors.New("orderNumber is required")
	ErrStatusRequired           = errors.New("status is required")
	ErrItemFieldsRequired       = errors.New("demandCode, description and deliveryDate are required")
	ErrInvalidDeliveryDate      = errors.New("invalid deliveryDate format, expected YYYY-MM-DD")
	ErrSyncNotCancellable       = errors.New("sync can only be cancelled while pending")
	ErrOrderClosed              = errors.New("order is closed and cannot be modified")
	ErrInvalidStatusTransition  = errors.New("invalid status transition")
	ErrInvalidOrderStatus       = errors.New("invalid order status")
	ErrInvalidItemStatus        = errors.New("invalid item status")
	ErrItemStatusImmutable      = errors.New("item status cannot be set directly")
	ErrItemNotSyncable          = errors.New("item must be UPDATED or FAILED to sync")
	ErrSyncAlreadyActive        = errors.New("item already has a pending or processing sync")
	ErrDeliveryDateInPast       = errors.New("deliveryDate cannot be in the past")
)
