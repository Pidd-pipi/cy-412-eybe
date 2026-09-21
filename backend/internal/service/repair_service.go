package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
)

// 业主撤销工单场景的哨兵错误，handler 据此映射为明确的 HTTP 状态与原因。
var (
	ErrRepairCancelForbidden   = errors.New("repair cancel forbidden: only creator can cancel")
	ErrRepairAlreadyCancelled  = errors.New("repair already cancelled")
	ErrRepairStatusAdvanced    = errors.New("repair status advanced past cancellable stage")
	ErrRepairCannotBeCancelled = errors.New("repair cannot be cancelled in current status")
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
		return model.Repair{}, fmt.Errorf("Repair[id=%d] assign failed: repair already cancelled, current role=%s: %w", id, role, ErrRepairAlreadyCancelled)
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
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if v.Status == constants.RepairStatusCancelled {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] status failed: repair already cancelled, current role=%s: %w", id, role, ErrRepairAlreadyCancelled)
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

// CancelByOwner 业主自行撤销报修工单。
// 规则：只有工单创建者本人能撤销；待受理/已分派可撤销；处理中及之后必须走完成/关闭流程；
// 物业与管理员不能代业主取消。重复取消返回明确原因。
func (s *RepairService) CancelByOwner(id, operatorID uint, role string) (model.Repair, error) {
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, fmt.Errorf("Repair[id=%d] cancel failed: %w", id, e)
	}
	// 越权取消：非创建者本人（含物业、管理员）一律拒绝。
	if v.UserID != operatorID {
		s.logger.Warn("repair cancel rejected: operator is not creator",
			"repair_id", id, "creator_id", v.UserID, "operator_id", operatorID, "operator_role", role)
		return model.Repair{}, fmt.Errorf("Repair[id=%d] cancel failed: operator user_id=%d role=%s is not creator user_id=%d: %w",
			id, operatorID, role, v.UserID, ErrRepairCancelForbidden)
	}
	switch v.Status {
	case constants.RepairStatusCancelled:
		// 重复取消。
		return model.Repair{}, fmt.Errorf("Repair[id=%d] cancel failed: repair already cancelled by user_id=%d, current role=%s: %w",
			id, operatorID, role, ErrRepairAlreadyCancelled)
	case constants.RepairStatusPending, constants.RepairStatusAssigned:
		// 允许业主撤销：条件更新防止“检查后状态被分派/推进”的并发竞态。
		cancellable := make([]string, 0, len(constants.RepairStatusCancellable))
		for st := range constants.RepairStatusCancellable {
			cancellable = append(cancellable, st)
		}
		affected, ce := s.repo.CancelByOwner(id, operatorID, cancellable, constants.RepairStatusCancelled)
		if ce != nil {
			return model.Repair{}, fmt.Errorf("Repair[id=%d] cancel failed: %w", id, ce)
		}
		if affected == 0 {
			latest, le := s.repo.ByID(id)
			if le == nil && latest.Status == constants.RepairStatusCancelled {
				return model.Repair{}, fmt.Errorf("Repair[id=%d] cancel failed: repair already cancelled by user_id=%d, current role=%s: %w",
					id, operatorID, role, ErrRepairAlreadyCancelled)
			}
			return model.Repair{}, fmt.Errorf("Repair[id=%d] cancel failed: status already advanced to %s, current role=%s: %w",
				id, latest.Status, role, ErrRepairStatusAdvanced)
		}
		s.logger.Info("repair cancelled by owner", "repair_id", id, "user_id", operatorID, "previous_status", v.Status)
		return s.repo.ByID(id)
	default:
		// processing / done / closed：状态已推进，必须走原有完成或关闭流程。
		return model.Repair{}, fmt.Errorf("Repair[id=%d] cancel failed: status %s cannot be cancelled, finish/close flow required, current role=%s: %w",
			id, v.Status, role, ErrRepairCannotBeCancelled)
	}
}

func (s *RepairService) OpenCount() (int64, error) { return s.repo.CountOpen() }
