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

func newRepairServiceFixture(t *testing.T) (*RepairService, model.User, model.User) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.User{}, &model.Repair{}); e != nil {
		t.Fatal(e)
	}
	owner := model.User{Phone: "1", Nickname: "owner", Role: constants.UserRoleResident}
	staff := model.User{Phone: "2", Nickname: "staff", Role: constants.UserRoleStaff}
	db.Create(&owner)
	db.Create(&staff)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRepairService(repository.NewRepairRepository(db), repository.NewUserRepository(db), logger), owner, staff
}

func TestCancelByOwnerTable(t *testing.T) {
	for _, tt := range []struct {
		name      string
		status    string
		operator  func(owner, staff model.User) (uint, string)
		seedTimes int // 同一工单预先取消的次数
		wantErr   error
	}{
		{"owner cancels pending", constants.RepairStatusPending, func(o, _ model.User) (uint, string) { return o.ID, o.Role }, 0, nil},
		{"owner cancels assigned", constants.RepairStatusAssigned, func(o, _ model.User) (uint, string) { return o.ID, o.Role }, 0, nil},
		{"owner cannot cancel processing", constants.RepairStatusProcessing, func(o, _ model.User) (uint, string) { return o.ID, o.Role }, 0, ErrRepairNotCancellable},
		{"owner cannot cancel done", constants.RepairStatusDone, func(o, _ model.User) (uint, string) { return o.ID, o.Role }, 0, ErrRepairNotCancellable},
		{"owner cannot cancel closed", constants.RepairStatusClosed, func(o, _ model.User) (uint, string) { return o.ID, o.Role }, 0, ErrRepairNotCancellable},
		{"repeat cancel rejected", constants.RepairStatusPending, func(o, _ model.User) (uint, string) { return o.ID, o.Role }, 1, ErrRepairAlreadyCancelled},
		{"staff cannot cancel for owner", constants.RepairStatusPending, func(_, s model.User) (uint, string) { return s.ID, s.Role }, 0, ErrRepairCancelForbidden},
		{"admin cannot cancel for owner", constants.RepairStatusAssigned, func(o, _ model.User) (uint, string) { return o.ID + 999, constants.UserRoleAdmin }, 0, ErrRepairCancelForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, owner, staff := newRepairServiceFixture(t)
			rp := model.Repair{UserID: owner.ID, Title: "灯坏了", Description: "客厅灯不亮需要检修", Type: "水电", Status: tt.status}
			if tt.status == constants.RepairStatusAssigned {
				rp.HandlerID = &staff.ID
			}
			v, e := svc.Create(owner.ID, rp.Title, rp.Description, rp.Type, "")
			if e != nil {
				t.Fatal(e)
			}
			id := v.ID
			if tt.status != constants.RepairStatusPending {
				if _, e = svc.UpdateStatus(id, tt.status, 0, constants.UserRoleStaff); e != nil {
					t.Fatal(e)
				}
			}
			for i := 0; i < tt.seedTimes; i++ {
				if _, e = svc.CancelByOwner(id, owner.ID, constants.UserRoleResident); e != nil {
					t.Fatal(e)
				}
			}
			opID, role := tt.operator(owner, staff)
			got, e := svc.CancelByOwner(id, opID, role)
			if tt.wantErr == nil {
				if e != nil {
					t.Fatalf("want success, got %v", e)
				}
				if got.Status != constants.RepairStatusCancelled || got.CancelledAt == nil {
					t.Fatalf("want cancelled with timestamp, got status=%s", got.Status)
				}
				open, _ := svc.OpenCount()
				if open != 0 {
					t.Fatalf("cancelled repair must not count as open, got %d", open)
				}
				return
			}
			if !errors.Is(e, tt.wantErr) {
				t.Fatalf("want err %v, got %v", tt.wantErr, e)
			}
		})
	}
}

// TestUpdateStatusRejectsCancelled 已取消工单不能再被分派/推进，
// cancelled 也不能由物业通过进度接口写入。
func TestUpdateStatusRejectsCancelled(t *testing.T) {
	svc, owner, _ := newRepairServiceFixture(t)
	v, e := svc.Create(owner.ID, "水管漏水", "厨房水管接口漏水", "水电", "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = svc.CancelByOwner(v.ID, owner.ID, constants.UserRoleResident); e != nil {
		t.Fatal(e)
	}
	if _, e = svc.UpdateStatus(v.ID, constants.RepairStatusProcessing, 0, constants.UserRoleStaff); !errors.Is(e, ErrRepairAlreadyCancelled) {
		t.Fatalf("update cancelled repair want ErrRepairAlreadyCancelled, got %v", e)
	}
	if _, e = svc.UpdateStatus(v.ID, constants.RepairStatusCancelled, 0, constants.UserRoleStaff); !errors.Is(e, ErrRepairCancelNotStaff) {
		t.Fatalf("staff writing cancelled want ErrRepairCancelNotStaff, got %v", e)
	}
}
