package handler

import (
	"errors"
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
		Fail(c, 500, constants.CodeInternal, e.Error())
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
		Fail(c, 500, constants.CodeInternal, e.Error())
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
		Fail(c, 400, constants.CodeBadRequest, e.Error())
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
		Fail(c, 400, constants.CodeBadRequest, e.Error())
		return
	}
	OK(c, v)
}

// Cancel 业主本人撤销报修工单；物业与管理员不能代业主取消。
func (h *RepairHandler) Cancel(c *gin.Context) {
	var r dto.CancelRepairRequest
	if c.Request.ContentLength > 0 && !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.CancelByOwner(uint(id), c.GetUint("userID"), c.GetString("role"))
	if e != nil {
		switch {
		case errors.Is(e, repository.ErrNotFound):
			Fail(c, 404, constants.CodeNotFound, e.Error())
		case errors.Is(e, service.ErrRepairCancelForbidden):
			// 越权取消（含物业、管理员代取消）。
			Fail(c, 403, constants.CodeForbidden, constants.MessageRepairCancelForbidden+": "+e.Error())
		case errors.Is(e, service.ErrRepairAlreadyCancelled):
			// 重复取消。
			Fail(c, 409, constants.CodeConflict, constants.MessageRepairAlreadyCanceled+": "+e.Error())
		case errors.Is(e, service.ErrRepairStatusAdvanced), errors.Is(e, service.ErrRepairCannotBeCancelled):
			// 状态已推进到处理中及之后，只能走完成/关闭流程。
			Fail(c, 409, constants.CodeConflict, e.Error())
		default:
			Fail(c, 500, constants.CodeInternal, e.Error())
		}
		return
	}
	OK(c, v)
}
