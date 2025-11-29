package shared

// BulkImportResult описывает результат пакетного импорта транзакций (для API и сервисов).
type BulkImportResult struct {
	Accepted int              `json:"accepted"`
	Rejected int              `json:"rejected"`
	Errors   []BulkImportError `json:"errors"`
}

// BulkImportError — ошибка для одной транзакции в batch по индексу и сообщению.
type BulkImportError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}
