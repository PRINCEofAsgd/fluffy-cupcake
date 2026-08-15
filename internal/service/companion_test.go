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
	inviterID  int64
	inviteeID  int64
	note       string
	endedAt    time.Time
	cutoff     time.Time
	confirmed  bool
	resolution string
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
func (f *fakeCompanionStore) RequestUnbind(_ context.Context, _ int64, _ int64, requestedAt, cutoff time.Time, confirmed bool) (model.UnbindAction, error) {
	f.endedAt, f.cutoff, f.confirmed = requestedAt, cutoff, confirmed
	if confirmed {
		return model.UnbindActionDirectlyEnded, nil
	}
	return model.UnbindActionRequestSent, nil
}
func (f *fakeCompanionStore) AcceptUnbind(context.Context, int64, int64, time.Time) error { return nil }
func (f *fakeCompanionStore) CancelUnbind(_ context.Context, _ int64, _ int64, respondedAt time.Time) error {
	f.endedAt, f.resolution = respondedAt, model.UnbindStatusCancelled
	return nil
}
func (f *fakeCompanionStore) RejectUnbind(_ context.Context, _ int64, _ int64, respondedAt time.Time) error {
	f.endedAt, f.resolution = respondedAt, model.UnbindStatusRejected
	return nil
}
func (f *fakeCompanionStore) ListPartners(context.Context, int64) ([]model.CompanionPartner, error) {
	return nil, nil
}

// TestResolveUnbindUsesServerTime 验证取消和拒绝都使用可信服务端时间写入反馈状态。
func TestResolveUnbindUsesServerTime(t *testing.T) {
	now := time.Date(2026, 8, 16, 13, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	for _, testCase := range []struct {
		name string
		want string
		call func(*CompanionService) error
	}{
		{name: "cancel", want: model.UnbindStatusCancelled, call: func(service *CompanionService) error {
			return service.CancelUnbind(context.Background(), 1, 9)
		}},
		{name: "reject", want: model.UnbindStatusRejected, call: func(service *CompanionService) error {
			return service.RejectUnbind(context.Background(), 2, 9)
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := &fakeCompanionStore{}
			service := NewCompanionService(store, fakeCompanionUsers{})
			service.now = func() time.Time { return now }
			if err := testCase.call(service); err != nil {
				t.Fatal(err)
			}
			if store.resolution != testCase.want || !store.endedAt.Equal(now.UTC()) {
				t.Fatalf("resolution=%q respondedAt=%v", store.resolution, store.endedAt)
			}
		})
	}
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

// TestRequestUnbindUsesThirtyDayCutoff 验证统一解绑入口传递服务端 30 天截止点和额外确认。
func TestRequestUnbindUsesThirtyDayCutoff(t *testing.T) {
	store := &fakeCompanionStore{}
	service := NewCompanionService(store, fakeCompanionUsers{})
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	action, err := service.RequestUnbind(context.Background(), 1, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	if action != model.UnbindActionDirectlyEnded || !store.endedAt.Equal(now) || !store.cutoff.Equal(now.AddDate(0, 0, -30)) || !store.confirmed {
		t.Fatalf("action=%q endedAt=%v cutoff=%v confirmed=%v", action, store.endedAt, store.cutoff, store.confirmed)
	}
}
