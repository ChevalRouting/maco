package api

import (
	"net/http"
	"strconv"
)

const defaultPageSize = 25

type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func pageParams(w http.ResponseWriter, r *http.Request) (page, pageSize int, ok bool) {
	page, pageSize = 1, defaultPageSize

	if value := r.URL.Query().Get("page"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid page")
			return 0, 0, false
		}

		page = n
	}

	if value := r.URL.Query().Get("page_size"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "page_size must be between 1 and 100")
			return 0, 0, false
		}

		pageSize = n
	}

	return page, pageSize, true
}

func paginate[T any](items []T, page, pageSize int) Page[T] {
	total := len(items)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}

	if start > total {
		start = total
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return Page[T]{
		Items:      items[start:end],
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
