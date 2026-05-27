package mysqlstore

import (
	"errors"
)

func (s *MySQLStore) CreateWallet(wallet Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := insertMySQLWallet(db, wallet); err != nil {
		if isUniqueConstraint(err) {
			return errAlreadyExists
		}
		return err
	}
	return nil
}

func (s *MySQLStore) UpdateWallet(wallet Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE wallets
SET name = ?, currency = ?, balance_initial = ?
WHERE id = ? AND user_id = ?`,
		wallet.Name,
		wallet.Currency,
		wallet.BalanceInitial,
		wallet.ID,
		wallet.UserID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func (s *MySQLStore) DeleteWallet(userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	var hasTx int
	err = db.QueryRow(`SELECT COUNT(1) FROM transactions WHERE wallet_id = ? AND deleted_at IS NULL`, id).Scan(&hasTx)
	if err != nil {
		return err
	}
	if hasTx > 0 {
		return errors.New("cannot delete wallet containing transactions")
	}

	var hasRecurring int
	err = db.QueryRow(`SELECT COUNT(1) FROM recurring_transactions WHERE wallet_id = ? AND deleted_at IS NULL`, id).Scan(&hasRecurring)
	if err != nil {
		return err
	}
	if hasRecurring > 0 {
		return errors.New("cannot delete wallet linked to recurring transactions")
	}

	result, err := db.Exec(`DELETE FROM wallets WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}
