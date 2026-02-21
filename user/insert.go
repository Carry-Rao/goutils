package user

import "time"

func (u *Users) Insert(user *User) error {
	stmt, err := u.db.Prepare("INSERT INTO users (group_id, username, password, email, phone, contact, encryte_token, status, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.GroupID, user.Username, user.Password, user.Email, user.Phone, user.Contact, user.EncryteToken, user.Status, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}
