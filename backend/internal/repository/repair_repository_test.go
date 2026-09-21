package repository

import (
	"testing"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRepairRepositoryListTable(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&model.User{}, &model.Repair{})
	u := model.User{Phone: "1", Nickname: "u", Role: "resident"}
	db.Create(&u)
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

func TestRepairRepositoryCancelByOwnerAndCountOpen(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&model.User{}, &model.Repair{})
	owner := model.User{Phone: "1", Nickname: "owner", Role: "resident"}
	other := model.User{Phone: "2", Nickname: "staff", Role: "staff"}
	db.Create(&owner)
	db.Create(&other)
	cancellable := []string{constants.RepairStatusPending, constants.RepairStatusAssigned}

	rp := model.Repair{UserID: owner.ID, Title: "A", Description: "d", Type: "水电", Status: constants.RepairStatusPending}
	db.Create(&rp)
	r := NewRepairRepository(db)

	// 非创建者无法通过条件更新撤销工单。
	if n, e := r.CancelByOwner(rp.ID, other.ID, cancellable, constants.RepairStatusCancelled); e != nil || n != 0 {
		t.Fatalf("cross-user cancel must affect 0 rows, got %d err %v", n, e)
	}
	// 创建者可撤销待受理工单。
	if n, e := r.CancelByOwner(rp.ID, owner.ID, cancellable, constants.RepairStatusCancelled); e != nil || n != 1 {
		t.Fatalf("owner cancel must affect 1 row, got %d err %v", n, e)
	}
	// 已取消工单无法重复撤销。
	if n, e := r.CancelByOwner(rp.ID, owner.ID, cancellable, constants.RepairStatusCancelled); e != nil || n != 0 {
		t.Fatalf("duplicate cancel must affect 0 rows, got %d err %v", n, e)
	}
	// 处理中工单不在可撤销状态集合内。
	adv := model.Repair{UserID: owner.ID, Title: "B", Description: "d", Type: "水电", Status: constants.RepairStatusProcessing}
	db.Create(&adv)
	if n, e := r.CancelByOwner(adv.ID, owner.ID, cancellable, constants.RepairStatusCancelled); e != nil || n != 0 {
		t.Fatalf("processing cancel must affect 0 rows, got %d err %v", n, e)
	}
	// 已取消工单不计入待处理数量，处理中的 adv 仍计入。
	n, e := r.CountOpen()
	if e != nil || n != 1 {
		t.Fatalf("open count want 1 (only processing), got %d err %v", n, e)
	}
}
