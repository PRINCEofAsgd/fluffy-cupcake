package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

// TestRequestUnbindRoutesByPartnerActivity 验证统一入口只在近期登录时发申请，并要求长期未登录的额外确认。
func TestRequestUnbindRoutesByPartnerActivity(t *testing.T) {
	now := time.Date(2026, 8, 16, 4, 0, 0, 0, time.UTC)
	cutoff := now.AddDate(0, 0, -30)

	t.Run("recent partner sends request", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		expectUnbindState(mock, 1, 9, now.Add(-time.Hour), model.UnbindStatusNone)
		mock.ExpectExec("UPDATE companion_bindings").
			WithArgs(int64(1), now, int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		action, err := NewCompanionRepository(db).RequestUnbind(context.Background(), 9, 1, now, cutoff, false)
		if err != nil || action != model.UnbindActionRequestSent {
			t.Fatalf("action=%q error=%v", action, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("cancelled request can be sent again", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		expectUnbindState(mock, 1, 9, now.Add(-time.Hour), model.UnbindStatusCancelled)
		mock.ExpectExec("UPDATE companion_bindings").
			WithArgs(int64(1), now, int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		action, err := NewCompanionRepository(db).RequestUnbind(context.Background(), 9, 1, now, cutoff, false)
		if err != nil || action != model.UnbindActionRequestSent {
			t.Fatalf("action=%q error=%v", action, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("inactive partner requires confirmation", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		expectUnbindState(mock, 1, 9, cutoff, model.UnbindStatusNone)
		mock.ExpectRollback()

		action, err := NewCompanionRepository(db).RequestUnbind(context.Background(), 9, 1, now, cutoff, false)
		if err != nil || action != model.UnbindActionInactiveConfirmationRequired {
			t.Fatalf("action=%q error=%v", action, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("confirmed inactive partner ends binding", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		expectUnbindState(mock, 1, 9, cutoff.Add(-time.Second), model.UnbindStatusNone)
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM companion_active_memberships WHERE binding_id = ?")).
			WithArgs(int64(9)).WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec("UPDATE companion_bindings").
			WithArgs(int64(1), now, model.UnbindStatusDirect, int64(1), now, now, int64(1), "inactive", int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		action, err := NewCompanionRepository(db).RequestUnbind(context.Background(), 9, 1, now, cutoff, true)
		if err != nil || action != model.UnbindActionDirectlyEnded {
			t.Fatalf("action=%q error=%v", action, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

// TestResolveUnbindRequestRoles 验证只有发起方能取消、只有接收方能拒绝，并写入可供收件箱展示的处理状态。
func TestResolveUnbindRequestRoles(t *testing.T) {
	now := time.Date(2026, 8, 16, 5, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		status string
		call   func(*CompanionRepository) error
	}{
		{
			name:   "requester cancels",
			status: model.UnbindStatusCancelled,
			call: func(repository *CompanionRepository) error {
				return repository.CancelUnbind(context.Background(), 9, 1, now)
			},
		},
		{
			name:   "recipient rejects",
			status: model.UnbindStatusRejected,
			call: func(repository *CompanionRepository) error {
				return repository.RejectUnbind(context.Background(), 9, 2, now)
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			userID := int64(1)
			if testCase.status == model.UnbindStatusRejected {
				userID = 2
			}
			mock.ExpectExec("UPDATE companion_bindings AS binding").
				WithArgs(userID, testCase.status, userID, now, int64(9), userID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			if err := testCase.call(NewCompanionRepository(db)); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("wrong role cannot resolve", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectExec("UPDATE companion_bindings AS binding").
			WithArgs(int64(2), model.UnbindStatusCancelled, int64(2), now, int64(9), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		err = NewCompanionRepository(db).CancelUnbind(context.Background(), 9, 2, now)
		if err != ErrInvalidBindingState {
			t.Fatalf("error=%v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

// expectUnbindState 建立统一解绑事务读取信件和对方最后登录时间的共同期望。
func expectUnbindState(mock sqlmock.Sqlmock, userID, bindingID int64, lastSeen time.Time, unbindStatus string) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT binding.inviter_id, binding.invitee_id").
		WithArgs(userID, bindingID).
		WillReturnRows(sqlmock.NewRows([]string{"inviter_id", "invitee_id", "status", "unbind_requested_by", "unbind_status"}).
			AddRow(int64(1), int64(2), "active", nil, unbindStatus))
	mock.ExpectQuery("SELECT COALESCE\\(last_login_at, created_at\\)").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"last_seen"}).AddRow(lastSeen))
}
