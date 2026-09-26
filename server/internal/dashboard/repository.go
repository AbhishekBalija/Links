package dashboard

import (
	"context"

	"gorm.io/gorm"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// Profile returns the user's display name and their Student identity's
// Department, if they have one.
func (r *GormRepository) Profile(ctx context.Context, userID string) (string, *string, error) {
	var row struct {
		FullName     string  `gorm:"column:full_name"`
		DepartmentID *string `gorm:"column:department_id"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT p.full_name, si.department_id
		FROM profiles p
		LEFT JOIN student_identities si ON si.user_id = p.user_id
		WHERE p.user_id = ?`, userID,
	).Scan(&row).Error
	return row.FullName, row.DepartmentID, err
}

func (r *GormRepository) Department(ctx context.Context, id string) (*Department, error) {
	var departments []Department
	err := r.db.WithContext(ctx).Raw(`SELECT id, code, name FROM departments WHERE id = ?`, id).Scan(&departments).Error
	if err != nil || len(departments) == 0 {
		return nil, err
	}
	return &departments[0], nil
}
