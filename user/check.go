package user

import "errors"

func (u *Users) Check(user User) (bool, error) {
	var count int
	if err := u.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (u *Users) CheckByUsernameAndPassword(username, password string) (User, error) {
	var user User
	if err := u.db.QueryRow("SELECT * FROM users WHERE username = ?", username).Scan(&user.ID, &user.GroupID, &user.Username, &user.Password, &user.Email, &user.Phone, &user.Contact, &user.EncryteToken, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}
	if user.Password != password {
		return User{}, errors.New("password is not correct")
	}
	return user, nil
}

func (u *Users) CheckStatus(id int) (bool, error) {
	var status int
	if err := u.db.QueryRow("SELECT status FROM users WHERE id = ?", id).Scan(&status); err != nil {
		return false, err
	}
	if status == 1 {
		return true, nil
	}
	return false, nil
}
