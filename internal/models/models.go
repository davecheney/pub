// package models contains the database models for the m.
// Urgh, a package called models, I know, I know.
package models

import "gorm.io/gorm"

// forEach runs each function in the slice within the supplied transaction.
func forEach(tx *gorm.DB, fns ...func(tx *gorm.DB) error) error {
	for _, fn := range fns {
		if err := fn(tx); err != nil {
			return err
		}
	}
	return nil
}

func withTransaction[P *T, T any](db *gorm.DB, fn func(tx *gorm.DB) (P, error)) (P, error) {
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	result, err := fn(tx)
	if tx.Error != nil || err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	return result, nil
}
