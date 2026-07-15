package domain

import "time"

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusOpen, OrderStatusInProgress, OrderStatusClosed:
		return true
	default:
		return false
	}
}

func (s ItemStatus) IsValid() bool {
	switch s {
	case ItemStatusPending, ItemStatusUpdated, ItemStatusSynced, ItemStatusFailed:
		return true
	default:
		return false
	}
}

func (s OrderStatus) CanTransitionTo(next OrderStatus, items []OrderItem) error {
	if !next.IsValid() {
		return ErrInvalidOrderStatus
	}
	if s == next {
		return nil
	}
	if s == OrderStatusClosed {
		return ErrOrderClosed
	}

	switch s {
	case OrderStatusOpen:
		if next == OrderStatusInProgress && len(items) > 0 {
			return nil
		}
	case OrderStatusInProgress:
		if next == OrderStatusClosed && AllItemsTerminal(items) {
			return nil
		}
	}

	return ErrInvalidStatusTransition
}

func AllItemsTerminal(items []OrderItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.Status != ItemStatusSynced && item.Status != ItemStatusFailed {
			return false
		}
	}
	return true
}

func (s ItemStatus) CanSync() bool {
	return s == ItemStatusUpdated || s == ItemStatusFailed
}

func (s ItemStatus) ShouldPromoteOnEdit() bool {
	return s == ItemStatusPending || s == ItemStatusSynced || s == ItemStatusFailed
}

func IsDeliveryDateInPast(date time.Time) bool {
	d := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return d.Before(today)
}
