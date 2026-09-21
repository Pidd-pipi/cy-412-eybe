package repository

import (
	"testing"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRepairDB(t *testing.T) (*gorm.DB, model.User) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.User{}, &model.Repair{}); e != nil {
		t.Fatal(e)
	}
	u := model.User{Phone: "1", Nickname: "u", Role: "resident"}
	if e = db.Create(&u).Error; e != nil {
		t.Fatal(e)

	}
	return db, u
}

func TestRepairRepositoryListTable(t *testing.T) {
	db, u := newTestRepairDB(t)
	db.Create(&model.Repair{UserID: u.ID, Title: "A", Description: "d", Type: "水电", Status: "pending"})
	r := NewRepairRepository(db)
	for _, tt := range []struct {
		status string
		want   int
	}{{"", 1}, {"pending", 1}, {"done", 0}, {"cancelled", 0}} {
		got, e := r.List(tt.status)
		if e != nil || len(got) != tt.want {
			t.Fatalf("status %s got %d err %v", tt.status, len(got), e)
		}
	}
}

// TestRepairRepositoryCancelTable 校验条件取消：仅 pending/assigned 可取消，
// 重复取消不命中，且取消后不计入待处理数量。
// 工单按 1=pending 2=assigned 3=processing 4=done 5=closed 的顺序创建。
func TestRepairRepositoryCancelTable(t *testing.T) {
	db, u := newTestRepairDB(t)
	for _, st := range []string{constants.RepairStatusPending, constants.RepairStatusAssigned, constants.RepairStatusProcessing, constants.RepairStatusDone, constants.RepairStatusClosed} {
		db.Create(&model.Repair{UserID: u.ID, Title: st, Description: "desc", Type: "水电", Status: st})
	}
	r := NewRepairRepository(db)
	allowed := []string{constants.RepairStatusPending, constants.RepairStatusAssigned}
	for _, tt := range []struct {
		id            uint
		wantCancelled bool
	}{{1, true}, {2, true}, {3, false}, {4, false}, {5, false}} {
		got, e := r.Cancel(tt.id, allowed, time.Now())
		if e != nil {
			t.Fatalf("cancel repair %d: %v", tt.id, e)
		}
		if got != tt.wantCancelled {
			t.Fatalf("cancel repair %d got %v want %v", tt.id, got, tt.wantCancelled)
		}
	}
	// 重复取消：已取消的工单再次条件更新不应命中
	got, e := r.Cancel(1, allowed, time.Now())
	if e != nil || got {
		t.Fatalf("repeat cancel got %v err %v, want false/nil", got, e)
	}
	v, e := r.ByID(1)
	if e != nil || v.Status != constants.RepairStatusCancelled || v.CancelledAt == nil {
		t.Fatalf("repair 1 after cancel: status=%s cancelled_at=%v err=%v", v.Status, v.CancelledAt, e)
	}
	// processing=1 条计入待处理；pending/assigned 已取消，done/closed 不计
	open, e := r.CountOpen()
	if e != nil || open != 1 {
		t.Fatalf("open count got %d err %v, want 1 (processing only)", open, e)
	}
}
