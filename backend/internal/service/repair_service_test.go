package service

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCancelTestDB(t *testing.T) (*gorm.DB, model.User, model.User) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.User{}, &model.Repair{}); e != nil {
		t.Fatal(e)
	}
	owner := model.User{Phone: "1", Nickname: "owner", Role: "resident"}
	staff := model.User{Phone: "2", Nickname: "staff", Role: "staff"}
	db.Create(&owner)
	db.Create(&staff)
	return db, owner, staff
}

func TestRepairCancelByOwnerTable(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     string
		operatorID func(owner, staff model.User) uint
		role       string
		wantStatus string
		wantErr    error
	}{
		{"owner cancels pending", constants.RepairStatusPending, func(o, _ model.User) uint { return o.ID }, "resident", constants.RepairStatusCancelled, nil},
		{"owner cancels assigned", constants.RepairStatusAssigned, func(o, _ model.User) uint { return o.ID }, "resident", constants.RepairStatusCancelled, nil},
		{"owner cannot cancel processing", constants.RepairStatusProcessing, func(o, _ model.User) uint { return o.ID }, "resident", "", ErrRepairCannotBeCancelled},
		{"owner cannot cancel done", constants.RepairStatusDone, func(o, _ model.User) uint { return o.ID }, "resident", "", ErrRepairCannotBeCancelled},
		{"owner cannot cancel closed", constants.RepairStatusClosed, func(o, _ model.User) uint { return o.ID }, "resident", "", ErrRepairCannotBeCancelled},
		{"staff cannot cancel for owner", constants.RepairStatusPending, func(_, s model.User) uint { return s.ID }, "staff", "", ErrRepairCancelForbidden},
		{"admin cannot cancel for owner", constants.RepairStatusAssigned, func(_, s model.User) uint { return s.ID }, "admin", "", ErrRepairCancelForbidden},
		{"duplicate cancel rejected", constants.RepairStatusCancelled, func(o, _ model.User) uint { return o.ID }, "resident", "", ErrRepairAlreadyCancelled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, owner, staff := newCancelTestDB(t)
			rp := model.Repair{UserID: owner.ID, Title: "灯坏了", Description: "客厅灯具闪烁需要检查", Type: "水电", Status: tt.status}
			if e := db.Create(&rp).Error; e != nil {
				t.Fatal(e)
			}
			svc := NewRepairService(repository.NewRepairRepository(db), repository.NewUserRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)))
			got, e := svc.CancelByOwner(rp.ID, tt.operatorID(owner, staff), tt.role)
			if tt.wantErr != nil {
				if !errors.Is(e, tt.wantErr) {
					t.Fatalf("want err %v, got %v", tt.wantErr, e)
				}
				return
			}
			if e != nil {
				t.Fatalf("unexpected err: %v", e)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("want status %s, got %s", tt.wantStatus, got.Status)
			}
			if got.UserID != owner.ID {
				t.Fatalf("cancelled repair record must be retained with original owner, got user_id=%d", got.UserID)
			}
		})
	}
}

func TestRepairCancelKeepsRecordAndExcludesFromOpenCount(t *testing.T) {
	db, owner, _ := newCancelTestDB(t)
	db.Create(&model.Repair{UserID: owner.ID, Title: "A", Description: "描述至少五个字", Type: "水电", Status: constants.RepairStatusPending})
	db.Create(&model.Repair{UserID: owner.ID, Title: "B", Description: "描述至少五个字", Type: "家具", Status: constants.RepairStatusCancelled})
	repo := repository.NewRepairRepository(db)
	svc := NewRepairService(repo, repository.NewUserRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)))

	n, e := svc.OpenCount()
	if e != nil || n != 1 {
		t.Fatalf("open count want 1, got %d err %v", n, e)
	}
	var pending model.Repair
	if e = db.Where("status = ?", constants.RepairStatusPending).First(&pending).Error; e != nil {
		t.Fatal(e)
	}
	v, e := svc.CancelByOwner(pending.ID, owner.ID, "resident")
	if e != nil || v.Status != constants.RepairStatusCancelled {
		t.Fatalf("cancel failed: %+v %v", v, e)
	}
	// 工单记录保留。
	if _, e = repo.ByID(pending.ID); e != nil {
		t.Fatalf("cancelled repair must remain in storage: %v", e)
	}
	// 不再计入待处理数量。
	n, e = svc.OpenCount()
	if e != nil || n != 0 {
		t.Fatalf("open count want 0 after cancel, got %d err %v", n, e)
	}
}
