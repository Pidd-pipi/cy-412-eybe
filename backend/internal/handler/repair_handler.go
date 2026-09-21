package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/repository"
	"github.com/smartestate/smartestate/internal/service"
)

type RepairHandler struct {
	Handler
	svc *service.RepairService
}

func NewRepairHandler(s *service.RepairService, h *Handler) *RepairHandler {
	return &RepairHandler{Handler: *h, svc: s}
}
func (h *RepairHandler) List(c *gin.Context) {
	v, e := h.svc.List(c.Query("status"))
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Create(c *gin.Context) {
	var r dto.CreateRepairRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	v, e := h.svc.Create(c.GetUint("userID"), r.Title, r.Description, r.Type, r.Images)
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Assign(c *gin.Context) {
	var r dto.AssignRepairRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Assign(uint(id), r.HandlerID, c.GetString("role"))
	if e != nil {
		if errors.Is(e, service.ErrRepairAlreadyCancelled) {
			Fail(c, http.StatusConflict, constants.CodeConflict, e.Error())
			return
		}
		Fail(c, 400, 40001, e.Error())
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Status(c *gin.Context) {
	var r dto.UpdateRepairStatusRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.UpdateStatus(uint(id), r.Status, r.Rating, c.GetString("role"))
	if e != nil {
		switch {
		case errors.Is(e, service.ErrRepairCancelNotStaff), errors.Is(e, service.ErrRepairAlreadyCancelled):
			Fail(c, http.StatusConflict, constants.CodeConflict, e.Error())
			return
		}
		Fail(c, 400, 40001, e.Error())
		return
	}
	OK(c, v)
}

// Cancel 业主撤销本人工单。只需登录态（Auth），不挂 repair:manage，
// 物业与管理员即使登录也会在服务层因非创建者被拒绝。
func (h *RepairHandler) Cancel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.CancelByOwner(uint(id), c.GetUint("userID"), c.GetString("role"))
	if e != nil {
		switch {
		case errors.Is(e, repository.ErrNotFound):
			Fail(c, http.StatusNotFound, constants.CodeNotFound, e.Error())
		case errors.Is(e, service.ErrRepairCancelForbidden):
			Fail(c, http.StatusForbidden, constants.CodeForbidden, e.Error())
		case errors.Is(e, service.ErrRepairAlreadyCancelled), errors.Is(e, service.ErrRepairNotCancellable), errors.Is(e, service.ErrRepairStatusAdvanced):
			Fail(c, http.StatusConflict, constants.CodeConflict, e.Error())
		default:
			Fail(c, http.StatusInternalServerError, constants.CodeInternal, e.Error())
		}
		return
	}
	OK(c, v)
}
