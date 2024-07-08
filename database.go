package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
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

	// 创建表
	createTable(db)
	//insertRecord(db, "Bob", "bob233", "bob@mines.com", 1)
	queryRecords(db)
	incrementAttr(db, 2, "medal")
	queryRecords(db)
	incrementAttr(db, 2, "medal")
	queryRecords(db)
}

func createTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT,
		name VARCHAR(50) NOT NULL,
		password VARCHAR(50) NOT NULL,
		email VARCHAR(50) NOT NULL,
		medal INT NOT NULL DEFAULT 0,
		PRIMARY KEY (id)
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Table created or already exists.")
}

func insertRecord(db *sql.DB, name string, password string, email string, medal int) {
	query := "INSERT INTO users (name, password, email, medal) VALUES (?, ?, ?, ?)"
	_, err := db.Exec(query, name, password, email, medal)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted record:", name, password, email, medal)
}
func addColumn(db *sql.DB) {
	query := "ALTER TABLE users ADD COLUMN qq VARCHAR(100);"
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added new column 'qq' to table 'users'.")
}
func dropColumn(db *sql.DB) {
	query := "ALTER TABLE users DROP COLUMN email;"
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Dropped column 'qq' from table 'users'.")
}

func queryRecords(db *sql.DB) {
	query := "SELECT id, name, password, medal FROM users"
	rows, err := db.Query(query)
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

func updateRecord(db *sql.DB, id int, newAge int) {
	query := "UPDATE users SET medal = ? WHERE id = ?"
	_, err := db.Exec(query, newAge, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated record id %d with new age %d\n", id, newAge)
}

func incrementAttr(db *sql.DB, id int, attr string) {
	query := fmt.Sprintf("UPDATE users SET %s = %s + 1 WHERE id = ?", attr, attr)
	_, err := db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Incremented %s for record id %d\n", attr, id)
}

func decrementAttr(db *sql.DB, id int, attr string) {
	query := "UPDATE users SET ? = ? - 1 WHERE id = ?"
	_, err := db.Exec(query, attr, attr, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decremented medal for record id %d\n", id)
}

//func deleteRecord(db *sql.DB, id int) {
//	query := "DELETE FROM users WHERE id = ?"
//	_, err := db.Exec(query, id)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Deleted record id %d\n", id)
//}
