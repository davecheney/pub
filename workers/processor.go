package workers

import (
	"time"

	"gorm.io/gorm"
)

// process makes one pass through the objects matching the scope, calling fn for each one.
// If fn returns an error, the object is updated with the error and the process continues.
// If fn returns nil, the object is deleted.
func process[T any](db *gorm.DB, scope func(*gorm.DB) *gorm.DB, fn func(*gorm.DB, T) error) error {
	var requests []T
	return db.Scopes(scope).FindInBatches(&requests, 100, func(db *gorm.DB, batch int) error {
		return forEach(requests, func(request T) error {
			start := time.Now()
			if err := fn(db, request); err != nil {
				return db.Model(request).UpdateColumns(map[string]interface{}{
					"attempts":     gorm.Expr("attempts + 1"),
					"last_attempt": start,
					"last_result":  err.Error(),
				}).Error
			}
			return db.Delete(request).Error
		})
	}).Error
}

func forEach[T any](a []T, fn func(T) error) error {
	for _, v := range a {
		if err := fn(v); err != nil {
			return err
		}
	}
	return nil
}
