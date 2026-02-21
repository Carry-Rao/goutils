package user

func (u *Users) Delete(id int) error {
	_, err := u.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}
