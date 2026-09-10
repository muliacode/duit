package store_test

import (
	"context"
	"testing"

	"duit/internal/domain"
	"duit/internal/store"
)

func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.OpenInMemory()
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestAccountRepository_CreateGetUpdateDelete(t *testing.T) {
	ctx := context.Background()
	repo := store.NewAccountRepository(newTestDB(t))

	created, err := repo.Create(
		ctx,
		domain.Account{Name: "Checking", Type: domain.AccountChecking, BalanceMinor: 10000},
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated ID")
	}
	if created.Kind() != domain.KindAsset {
		t.Fatalf("expected asset kind, got %s", created.Kind())
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.BalanceMinor != 10000 {
		t.Fatalf("expected balance 10000, got %d", got.BalanceMinor)
	}

	got.BalanceMinor = 5000
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, _ := repo.Get(ctx, created.ID)
	if updated.BalanceMinor != 5000 {
		t.Fatalf("expected updated balance 5000, got %d", updated.BalanceMinor)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, created.ID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestAccountRepository_DeleteMissing(t *testing.T) {
	repo := store.NewAccountRepository(newTestDB(t))
	if err := repo.Delete(context.Background(), "does-not-exist"); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAccountType_Kind(t *testing.T) {
	cases := map[domain.AccountType]domain.AccountKind{
		domain.AccountChecking:   domain.KindAsset,
		domain.AccountCreditCard: domain.KindLiability,
		domain.AccountMortgage:   domain.KindLiability,
		domain.AccountProperty:   domain.KindAsset,
	}
	for typ, want := range cases {
		if got := typ.Kind(); got != want {
			t.Errorf("%s.Kind() = %s, want %s", typ, got, want)
		}
	}
}
