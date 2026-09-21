package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
	"time"
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

// Cancel 在数据库侧做条件更新：仅当工单仍为可撤销状态且未取消时生效，
// 防止读取后状态被推进导致的并发覆盖。返回 rows==false 表示条件未满足。
func (r *RepairRepository) Cancel(id uint, statuses []string, at time.Time) (bool, error) {
	res := r.DB.Model(&model.Repair{}).
		Where("id = ? AND status IN ? AND cancelled_at IS NULL", id, statuses).
		Updates(map[string]any{"status": constants.RepairStatusCancelled, "cancelled_at": at})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// CountOpen 待处理数量不包含已完成、已关闭与业主已取消的工单。
func (r *RepairRepository) CountOpen() (int64, error) {
	var n int64
	e := r.DB.Model(&model.Repair{}).Where("status NOT IN ?", []string{constants.RepairStatusDone, constants.RepairStatusClosed, constants.RepairStatusCancelled}).Count(&n).Error
	return n, e
}
