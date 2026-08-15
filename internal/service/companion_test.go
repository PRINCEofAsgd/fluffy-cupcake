package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
)

type fakeCompanionStore struct {
	inviterID int64
	inviteeID int64
	note      string
	endedAt   time.Time
	cutoff    time.Time
}

func (f *fakeCompanionStore) GetActiveState(context.Context, int64) (model.CompanionState, error) {
	return model.CompanionState{}, nil
}
func (f *fakeCompanionStore) CreateInvitation(_ context.Context, inviterID, inviteeID int64, note string) (int64, error) {
	f.inviterID, f.inviteeID, f.note = inviterID, inviteeID, note
	return 11, nil
}
func (f *fakeCompanionStore) AcceptInvitation(context.Context, int64, int64, time.Time) error {
	return nil
}
func (f *fakeCompanionStore) ListInbox(context.Context, int64) ([]model.CompanionBinding, error) {
	return nil, nil
}
func (f *fakeCompanionStore) UpdateNote(context.Context, int64, int64, string) error { return nil }
func (f *fakeCompanionStore) RequestUnbind(context.Context, int64, int64, time.Time) error {
	return nil
}
func (f *fakeCompanionStore) AcceptUnbind(context.Context, int64, int64, time.Time) error { return nil }
func (f *fakeCompanionStore) DirectUnbind(_ context.Context, _ int64, _ int64, endedAt, cutoff time.Time) error {
	f.endedAt, f.cutoff = endedAt, cutoff
	return nil
}
func (f *fakeCompanionStore) ListPartners(context.Context, int64) ([]model.CompanionPartner, error) {
	return nil, nil
}

type fakeCompanionUsers struct{ users map[string]model.User }

func (f fakeCompanionUsers) GetByUsername(_ context.Context, username string) (model.User, error) {
	user, ok := f.users[username]
	if !ok {
		return model.User{}, repository.ErrNotFound
	}
	return user, nil
}

// TestCompanionInviteValidation 验证邀请解析用户名、禁止绑定自己并限制 Unicode 备注长度。
func TestCompanionInviteValidation(t *testing.T) {
	store := &fakeCompanionStore{}
	service := NewCompanionService(store, fakeCompanionUsers{users: map[string]model.User{"partner": {ID: 2, Username: "partner"}, "self": {ID: 1, Username: "self"}}})
	id, err := service.Invite(context.Background(), 1, " partner ", "  亲爱的  ")
	if err != nil || id != 11 || store.inviterID != 1 || store.inviteeID != 2 || store.note != "亲爱的" {
		t.Fatalf("Invite() id=%d error=%v store=%#v", id, err, store)
	}
	if _, err := service.Invite(context.Background(), 1, "self", ""); err != ErrCannotBindSelf {
		t.Fatalf("绑定自己 error=%v", err)
	}
	if _, err := service.Invite(context.Background(), 1, "partner", strings.Repeat("想", 65)); err != ErrCompanionNoteTooLong {
		t.Fatalf("超长备注 error=%v", err)
	}
}

// TestDirectUnbindUsesThirtyDayCutoff 验证直接解绑严格使用服务端当前时间减 30 天。
func TestDirectUnbindUsesThirtyDayCutoff(t *testing.T) {
	store := &fakeCompanionStore{}
	service := NewCompanionService(store, fakeCompanionUsers{})
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	if err := service.DirectUnbind(context.Background(), 1, 9); err != nil {
		t.Fatal(err)
	}
	if !store.endedAt.Equal(now) || !store.cutoff.Equal(now.AddDate(0, 0, -30)) {
		t.Fatalf("endedAt=%v cutoff=%v", store.endedAt, store.cutoff)
	}
}
