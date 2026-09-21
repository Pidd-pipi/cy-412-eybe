package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

type RepairRepository struct{ DB *gorm.DB }

func NewRepairRepository(db *gorm.DB) *RepairRepository  { return &RepairRepository{db} }
func (r *RepairRepository) Create(v *model.Repair) error { return r.DB.Create(v).Error }
func (r *RepairRepository) List(status string) (out []model.Repair, e error) {
	q := r.DB.Preload("User").Preload("Handler").Order("created_at desc")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	e = q.Find(&out).Error
	return
}
func (r *RepairRepository) ByID(id uint) (v model.Repair, e error) {
	e = r.DB.Preload("User").Preload("Handler").First(&v, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = ErrNotFound
	}
	return
}
func (r *RepairRepository) Update(v *model.Repair) error { return r.DB.Save(v).Error }

// CancelByOwner 仅当工单仍处于 fromStatuses 中的状态时，原子地把工单置为 toStatus。
// 返回 rowsAffected：0 表示状态已被推进（或已取消），调用方应重新读取并返回明确原因。
func (r *RepairRepository) CancelByOwner(id, userID uint, fromStatuses []string, toStatus string) (int64, error) {
	res := r.DB.Model(&model.Repair{}).
		Where("id = ? AND user_id = ? AND status IN ?", id, userID, fromStatuses).
		Update("status", toStatus)
	return res.RowsAffected, res.Error
}

func (r *RepairRepository) CountOpen() (int64, error) {
	var n int64
	// 已完成、已关闭、已取消均不再计入待处理数量。
	e := r.DB.Model(&model.Repair{}).Where("status NOT IN ?", []string{"done", "closed", "cancelled"}).Count(&n).Error
	return n, e
}
