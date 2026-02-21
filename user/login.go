package user

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Carry-Rao/goutils/encoding"
	"github.com/Carry-Rao/goutils/net"
)

func (u *Users) login(email, password string) (User, error) {
	user, err := u.SearchByEmail(email)
	if err != nil {
		return User{}, err
	}
	if user.Password != password {
		return User{}, errors.New("password is not correct")
	}
	return user, nil
}

func (u *Users) Login(w http.ResponseWriter, r *http.Request) {
	data := net.ParseData(r)
	email, ok := data["email"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	user, err := u.SearchByEmail(email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	session := encoding.RandomString(16)
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   session,
		Expires: time.Now().Add(time.Hour * 1),
	})
	u.kv.Set(context.Background(), session, user.ID, time.Hour*1)
}

func (u *Users) LoginSession(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie("session")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID, err := u.kv.Get(context.Background(), session.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := u.SearchByID(userID.(int))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err = u.login(user.Email, user.Password)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   session.Value,
		Expires: time.Now().Add(time.Hour * 24),
	})
}
