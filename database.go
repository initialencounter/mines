package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// DBHandler 是一个封装了数据库操作的结构体
type DBHandler struct {
	db *sql.DB
}

// NewDBHandler 创建一个新的 DBHandler 实例
func NewDBHandler(dsn string) (*DBHandler, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &DBHandler{db: db}, nil
}

func test() {
	dsn := "root:mines@tcp(localhost:3306)/mines"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	// 确认连接有效
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func (handler *DBHandler) createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT,
		name VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(50) NOT NULL,
		email VARCHAR(50) NOT NULL,
		medal INT NOT NULL DEFAULT 0,
		PRIMARY KEY (id)
	);`
	_, err := handler.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Table created or already exists.")
}

func (handler *DBHandler) insertRecord(name string, password string, email string, medal int) {
	query := "INSERT INTO users (name, password, email, medal) VALUES (?, ?, ?, ?)"
	_, err := handler.db.Exec(query, name, password, email, medal)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted record:", name, password, email, medal)
}
func (handler *DBHandler) addColumn() {
	query := "ALTER TABLE users ADD COLUMN qq VARCHAR(100);"
	_, err := handler.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added new column 'qq' to table 'users'.")
}
func (handler *DBHandler) dropColumn() {
	query := "ALTER TABLE users DROP COLUMN email;"
	_, err := handler.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Dropped column 'qq' from table 'users'.")
}

func (handler *DBHandler) queryRecords() {
	query := "SELECT id, name, password, medal FROM users"
	rows, err := handler.db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	fmt.Println("Current records:")
	for rows.Next() {
		var id int
		var name string
		var password string
		var medal int
		err := rows.Scan(&id, &name, &password, &medal)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name, password, medal)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func (handler *DBHandler) updateMedal(id int, newMedal int) {
	query := "UPDATE users SET medal = ? WHERE id = ?"
	_, err := handler.db.Exec(query, newMedal, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated record id %d with new age %d\n", id, newMedal)
}

func (handler *DBHandler) incrementAttr(id int, attr string) {
	allowedAttrs := map[string]bool{
		"medal": true,
	}

	// Check if the attr is allowed
	if !allowedAttrs[attr] {
		log.Fatalf("Invalid attribute: %s", attr)
		return
	}
	// language=ignore
	query := fmt.Sprintf("UPDATE users SET %s = %s + 1 WHERE id = ?", attr, attr)
	_, err := handler.db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Incremented %s for record id %d\n", attr, id)
}

func (handler *DBHandler) decrementAttr(id int, attr string) {
	allowedAttrs := map[string]bool{
		"medal": true,
	}

	// Check if the attr is allowed
	if !allowedAttrs[attr] {
		log.Fatalf("Invalid attribute: %s", attr)
		return
	}
	query := "UPDATE users SET ? = ? - 1 WHERE id = ?"
	_, err := handler.db.Exec(query, attr, attr, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decremented medal for record id %d\n", id)
}

func (handler *DBHandler) deleteRecord(id int) {
	query := "DELETE FROM users WHERE id = ?"
	_, err := handler.db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deleted record id %d\n", id)
}

func (handler *DBHandler) changePassword(id int, newPassword string) {
	query := "UPDATE users SET password = ? WHERE id = ?"
	_, err := handler.db.Exec(query, newPassword, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated password for record id %d\n", id)
}

func (handler *DBHandler) nameExists(name string) bool {
	query := "SELECT COUNT(*) FROM users WHERE name = ?"
	var count int
	err := handler.db.QueryRow(query, name).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count > 0
}

// Close 关闭数据库连接
func (handler *DBHandler) Close() {
	err := handler.db.Close()
	if err != nil {
		return
	}
}
