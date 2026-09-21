package service

import (
	"errors"
	"fmt"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"log/slog"
	"time"
)

// 业主撤销工单的哨兵错误，handler 层用 errors.Is 映射 HTTP 状态与业务码。
var (
	ErrRepairCancelForbidden  = errors.New(constants.ReasonRepairCancelForbidden)
	ErrRepairAlreadyCancelled = errors.New(constants.ReasonRepairAlreadyCancelled)
	ErrRepairNotCancellable   = errors.New(constants.ReasonRepairNotCancellable)
	ErrRepairStatusAdvanced   = errors.New(constants.ReasonRepairStatusAdvanced)
	ErrRepairCancelNotStaff   = errors.New(constants.ReasonRepairCancelNotStaff)
)

type RepairService struct {
	repo   *repository.RepairRepository
	users  *repository.UserRepository
	logger *slog.Logger
}

func NewRepairService(r *repository.RepairRepository, u *repository.UserRepository, l *slog.Logger) *RepairService {
	return &RepairService{r, u, l}
}
func (s *RepairService) Create(uid uint, title, desc, typ, images string) (model.Repair, error) {
	v := model.Repair{UserID: uid, Title: title, Description: desc, Type: typ, Images: images, Status: constants.RepairStatusPending}
	if e := s.repo.Create(&v); e != nil {
		return v, fmt.Errorf("Repair[user_id=%d] create failed: %w", uid, e)
	}
	return s.repo.ByID(v.ID)
}
func (s *RepairService) List(status string) ([]model.Repair, error) { return s.repo.List(status) }
func (s *RepairService) Assign(id, handlerID uint, role string) (model.Repair, error) {
	handler, e := s.users.ByID(handlerID)
	if e != nil {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] assign failed: staff not found, current role=%s: %w", id, role, e)
	}
	if handler.Role != constants.UserRoleStaff && handler.Role != constants.UserRoleAdmin {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] assign failed: handler %d is not staff/admin, current role=%s", id, handlerID, role)
	}
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if v.Status == constants.RepairStatusCancelled {
		return v, fmt.Errorf("Repair[id=%d] assign failed: %w, current role=%s", id, ErrRepairAlreadyCancelled, role)
	}
	v.HandlerID = &handlerID
	v.Handler = nil
	v.User = model.User{}
	v.Status = constants.RepairStatusAssigned
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] assign failed: %w", id, e)
	}
	return s.repo.ByID(id)
}
func (s *RepairService) UpdateStatus(id uint, status string, rating int, role string) (model.Repair, error) {
	if !constants.ValidRepairStatuses[status] {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] status failed: invalid status, current role=%s", id, role)
	}
	// cancelled 是业主专属终态，物业与管理员不能通过进度接口代业主取消。
	if status == constants.RepairStatusCancelled {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] status failed: %w, current role=%s", id, ErrRepairCancelNotStaff, role)
	}
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if v.Status == constants.RepairStatusCancelled {
		return v, fmt.Errorf("Repair[id=%d] status failed: %w, current role=%s", id, ErrRepairAlreadyCancelled, role)
	}
	v.Status = status
	if rating > 0 {
		v.Rating = rating
	}
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] status failed: %w", id, e)
	}
	return s.repo.ByID(id)
}

// CancelByOwner 业主撤销本人报修工单的闭环：
// 仅创建者本人可操作；仅 pending/assigned 可取消；处理中及之后必须走完成/关闭流程。
// 重复取消、越权取消、状态已推进都会返回带有明确原因的哨兵错误。
func (s *RepairService) CancelByOwner(id, operatorID uint, role string) (model.Repair, error) {
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, fmt.Errorf("Repair[id=%d] cancel failed: %w, current role=%s", id, e, role)
	}
	if v.UserID != operatorID {
		return v, fmt.Errorf("Repair[id=%d] cancel failed: operator user_id=%d is not the owner user_id=%d, current role=%s: %w", id, operatorID, v.UserID, role, ErrRepairCancelForbidden)
	}
	if v.Status == constants.RepairStatusCancelled {
		return v, fmt.Errorf("Repair[id=%d] cancel failed by user_id=%d, current role=%s: %w", id, operatorID, role, ErrRepairAlreadyCancelled)
	}
	if !constants.OwnerCancellableStatuses[v.Status] {
		return v, fmt.Errorf("Repair[id=%d] cancel failed: current status=%s, current role=%s: %w", id, v.Status, role, ErrRepairNotCancellable)
	}
	allowed := make([]string, 0, len(constants.OwnerCancellableStatuses))
	for st := range constants.OwnerCancellableStatuses {
		allowed = append(allowed, st)
	}
	ok, e := s.repo.Cancel(id, allowed, time.Now())
	if e != nil {
		return v, fmt.Errorf("Repair[id=%d] cancel failed: %w, current role=%s", id, e, role)
	}
	if !ok {
		// 条件更新未命中：读取后状态已被分派/处理流程推进，属于并发推进。
		return v, fmt.Errorf("Repair[id=%d] cancel failed by user_id=%d, current role=%s: %w", id, operatorID, role, ErrRepairStatusAdvanced)
	}
	s.logger.Info("repair cancelled by owner", "repair_id", id, "user_id", operatorID)
	return s.repo.ByID(id)
}
func (s *RepairService) OpenCount() (int64, error) { return s.repo.CountOpen() }
