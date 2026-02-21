package user

func (u *Users) SearchByID(id int) (User, error) {
	var user User
	if err := u.db.QueryRow("SELECT * FROM users WHERE id = ?", id).Scan(&user.ID, &user.GroupID, &user.Username, &user.Password, &user.Email, &user.Phone, &user.Contact, &user.EncryteToken, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}
	return user, nil
}

func (u *Users) SearchByEmail(email string) (User, error) {
	var user User
	if err := u.db.QueryRow("SELECT * FROM users WHERE email = ?", email).Scan(&user.ID, &user.GroupID, &user.Username, &user.Password, &user.Email, &user.Phone, &user.Contact, &user.EncryteToken, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}
	return user, nil
}
