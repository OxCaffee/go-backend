package data

import (
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"log"
)

var Db *sql.DB

// 初始化数据库
func init() {
	var err error
	Db, err = sql.Open("postgres", "dbname=chitchat sslmod=disable")
	if err != nil {
		log.Fatal(err)
	}
}

// 创建uuid
func createUUID() (uuid string) {
	u := new([16]byte)
	_, err := rand.Read(u[:])
	if err != nil {
		log.Fatal("Cannot generate UUID", err)
	}

	// 0x40 为 RFC4122 保留变量
	u[8] = (u[8] | 0x40) & 0x7f
	// 设置最重要的12-15比特位time和version相关
	u[6] = (u[6] & 0xf) | (0x4 << 4)
	uuid = fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
	return
}

// 哈希明文SHA-1加密
func Encrypt(plainText string) (cryptText string) {
	cryptText = fmt.Sprintf("%x", sha1.Sum([]byte(plainText)))
	return cryptText
}
