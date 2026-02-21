package user

import "time"

func (u *Users) Update(user *User) error {
	stmt, err := u.db.Prepare("UPDATE users SET group_id=?, username=?, password=?, email=?, phone=?, contact=?, encryte_token=?, status=?, updated_at=? WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(user.GroupID, user.Username, user.Password, user.Email, user.Phone, user.Contact, user.EncryteToken, user.Status, time.Now().Unix(), user.ID)
	return err
}
