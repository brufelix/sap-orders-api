package domain

type OrderListFilter struct {
	Page   int
	Limit  int
	Status *OrderStatus
}

type PagedResult[T any] struct {
	Data       []T   `json:"data"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

func NewPagedResult[T any](data []T, page, limit int, total int64) *PagedResult[T] {
	if data == nil {
		data = []T{}
	}

	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &PagedResult[T]{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
