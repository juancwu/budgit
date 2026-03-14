package service

// Paginate calculates pagination values from a page number, total count, and page size.
// Returns the adjusted page, total pages, and offset for the query.
func Paginate(page, total, perPage int) (adjustedPage, totalPages, offset int) {
	totalPages = (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset = (page - 1) * perPage
	return page, totalPages, offset
}
