package data

import (
	"time"
)

type User struct {
	Id        int
	Uuid      string
	Name      string
	Email     string
	Password  string
	CreatedAt time.Time
}

type Session struct {
	Id        int
	Uuid      string
	Email     string
	UserId    int
	CreatedAt time.Time
}

// 为已经存在的User创建一个新的session
func (user *User) CreateSession() (session Session, err error) {
	statement := "INSERT INTO sessions(uuid, email, user_id, created_at) VALUES ($1, $2, $3, $4) RETURNING id, uuid, email, user_id, created_at"
	stmt, err := Db.Prepare(statement)
	if err != nil {
		return
	}
	defer stmt.Close()
	err = stmt.QueryRow(createUUID(), user.Email, user.Id, time.Now()).Scan(&session.Id, &session.Uuid, &session.Email, &session.UserId, &session.CreatedAt)
	return
}

// 根据User获取已经存在的session
func (user *User) Session() (session Session, err error) {
	session = Session{}
	err = Db.QueryRow("SELECT id, uuid, email, user_id, created_at FROM sessions WHERE user_id = $1", user.Id).Scan(
		&session.Id, &session.Uuid, &session.Email, &session.UserId, &session.CreatedAt)
	return
}

// 根据数据库检查session是否有效
func (session *Session) Check() (valid bool, err error) {
	err = Db.QueryRow("SELECT id, uuid, email, user_id, created_at FROM sessions WHERE uuid = $1", session.Uuid).Scan(
		&session.Id, &session.Uuid, &session.Email, &session.UserId, &session.CreatedAt)
	if err != nil {
		valid = false
		return
	}
	if session.Id != 0 {
		valid = true
	}
	return
}

// 根据uuid删除session
func (session *Session) DeleteByUuid() (err error) {
	statement := "DELETE FROM sessions WHERE uuid = $1"
	stmt, err := Db.Prepare(statement)
	defer stmt.Close()

	if err != nil {
		return
	}

	_, err = Db.Exec(session.Uuid)
	return
}

// 根据session中的user id去寻找user
func (session *Session) User() (user User, err error) {
	statement := "SELECT id, uuid, name, email created_at FROM users WHERE id = $1"
	user = User{}
	err = Db.QueryRow(statement, session.UserId).Scan(&user.Id, &user.Uuid, &user.Name, &user.Email, &user.CreatedAt)
	return
}

// 删除所有的session
func (session *Session) DeleteAllSession() (err error) {
	statement := "DELETE FROM sessions"
	_, err = Db.Exec(statement)
	return
}

// 创建一个新的User并保存User的信息到数据库当中
func (user *User) CreateUser() (err error) {
	statement := "INSERT INTO users (uuid, name, email, password, created_at) VALUES ($1, $2, $3, $4, $5) returning id, uuid, created_at"
	stmt, err := Db.Prepare(statement)

	if err != nil {
		return
	}

	defer stmt.Close()

	err = Db.QueryRow(createUUID(), user.Name, user.Email, Encrypt(user.Password), time.Now()).Scan(&user.Id, &user.Uuid, &user.CreatedAt)
	return
}

// 从数据库中删除用户
func (user *User) DeleteUser() (err error) {
	statement := "DELETE FROM users WHERE id = $1"
	stmt, err := Db.Prepare(statement)

	if err != nil {
		return
	}

	defer stmt.Close()

	_, err = stmt.Exec(user.Id)
	return
}

// 更新数据库中的用户信息
func (user *User) UpdateUser() (err error) {
	statement := "UPDATE users SET name = $2, email = $3 WHERE id = $1"
	stmt, err := Db.Prepare(statement)

	if err != nil {
		return
	}

	defer stmt.Close()
	_, err = stmt.Exec(user.Id, user.Name, user.Email)
	return
}

// 删除所有的用户
func (user *User) DeleteAllUsers() (err error) {
	statement := "DELETE FROM users"
	_, err = Db.Exec(statement)
	return
}

// 获得所有的用户信息
func (user *User) AllUsers() (users []User, err error) {
	users = make([]User, 0)
	rows, err := Db.Query("SELECT id, uuid, name, email, password, created_at FROM users")

	if err != nil {
		return
	}

	for rows.Next() {
		user := User{}
		if err = rows.Scan(&user.Id, &user.Uuid, &user.Name, &user.Email, &user.Password, &user.CreatedAt); err != nil {
			return
		}
		users = append(users, user)
	}
	rows.Close()
	return
}

func UserByEmail(email string) (user User, err error) {
	user = User{}
	err = Db.QueryRow("SELECT id, uuid, name, email, password, created_at FROM users WHERE email = $1", email).
		Scan(&user.Id, &user.Uuid, &user.Name, &user.Email, &user.Password, &user.CreatedAt)
	return
}

func UserByUUID(uuid string) (user User, err error) {
	user = User{}
	err = Db.QueryRow("SELECT id, uuid, name, email, password, created_at FROM users WHERE uuid = $1", uuid).
		Scan(&user.Id, &user.Uuid, &user.Name, &user.Email, &user.Password, &user.CreatedAt)
	return
}
