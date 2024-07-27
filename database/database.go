package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// DBHandler 是一个封装了数据库操作的结构体
type DBHandler struct {
	Db *sql.DB
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

	return &DBHandler{Db: db}, nil
}

func (handler *DBHandler) CreateTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT,
		name VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(50) NOT NULL,
		email VARCHAR(50) NOT NULL,
		medal INT NOT NULL DEFAULT 0,
		PRIMARY KEY (id)
	);`
	_, err := handler.Db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Table created or already exists.")
}

func (handler *DBHandler) InsertRecord(name string, password string, email string, medal int) error {
	query := "INSERT INTO users (name, password, email, medal) VALUES (?, ?, ?, ?)"
	_, err := handler.Db.Exec(query, name, password, email, medal)
	if err != nil {
		log.Fatal(err)
	}
	return err
}
func (handler *DBHandler) AddColumn() error {
	query := "ALTER TABLE users ADD COLUMN qq VARCHAR(100);"
	_, err := handler.Db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added new column 'qq' to table 'users'.")
	return err
}
func (handler *DBHandler) DropColumn() error {
	query := "ALTER TABLE users DROP COLUMN email;"
	_, err := handler.Db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Dropped column 'qq' from table 'users'.")
	return err
}

func (handler *DBHandler) QueryRecords() error {
	query := "SELECT id, name, password, medal FROM users"
	rows, err := handler.Db.Query(query)
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
	return err
}

func (handler *DBHandler) UpdateMedal(id int, newMedal int) error {
	query := "UPDATE users SET medal = ? WHERE id = ?"
	_, err := handler.Db.Exec(query, newMedal, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated record id %d with new age %d\n", id, newMedal)
	return err

}

func (handler *DBHandler) IncrementAttr(id int, attr string) error {
	allowedAttrs := map[string]bool{
		"medal": true,
	}

	// Check if the attr is allowed
	if !allowedAttrs[attr] {
		log.Fatalf("Invalid attribute: %s", attr)
		return nil
	}
	// language=ignore
	query := fmt.Sprintf("UPDATE users SET %s = %s + 1 WHERE id = ?", attr, attr)
	_, err := handler.Db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Incremented %s for record id %d\n", attr, id)
	return err
}

func (handler *DBHandler) DecrementAttr(id int, attr string) error {
	allowedAttrs := map[string]bool{
		"medal": true,
	}

	// Check if the attr is allowed
	if !allowedAttrs[attr] {
		log.Fatalf("Invalid attribute: %s", attr)
		return nil
	}
	query := "UPDATE users SET ? = ? - 1 WHERE id = ?"
	_, err := handler.Db.Exec(query, attr, attr, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decremented medal for record id %d\n", id)
	return err
}

func (handler *DBHandler) AddMedal(id int, score int) error {
	if score == 0 {
		return nil
	}
	var query = "UPDATE users SET medal = medal + ? WHERE id = ?"
	if score < 0 {
		score = -score
		query = "UPDATE users SET medal = medal - ? WHERE id = ?"
	}
	_, err := handler.Db.Exec(query, score, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decremented medal for record id %d\n", id)
	return err
}

func (handler *DBHandler) DeleteRecord(id int) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := handler.Db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deleted record id %d\n", id)
	return err
}

func (handler *DBHandler) ChangePassword(id int, newPassword string) error {
	query := "UPDATE users SET password = ? WHERE id = ?"
	_, err := handler.Db.Exec(query, newPassword, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated password for record id %d\n", id)
	return err
}

func (handler *DBHandler) NameExists(name string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE name = ?"
	var count int
	err := handler.Db.QueryRow(query, name).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count > 0, err
}

func (handler *DBHandler) PasswordMatch(name string, password string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE name = ? AND password = ?"
	var count int
	err := handler.Db.QueryRow(query, name, password).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count > 0, err
}

func (handler *DBHandler) GetId(name string) (int, error) {
	query := "SELECT id FROM users WHERE name = ?"
	var id int
	err := handler.Db.QueryRow(query, name).Scan(&id)
	return id, err
}

func (handler *DBHandler) GetName(id int) (string, error) {
	query := "SELECT name FROM users WHERE id = ?"
	var name string
	err := handler.Db.QueryRow(query, id).Scan(&name)
	return name, err
}

// Close 关闭数据库连接
func (handler *DBHandler) Close() {
	err := handler.Db.Close()
	if err != nil {
		return
	}
}
