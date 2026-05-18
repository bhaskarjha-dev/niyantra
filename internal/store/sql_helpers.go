package store

func nullableAccountID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}
