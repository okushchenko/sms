package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/alexgear/sms/common"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var err error

func InitDB(dbname string) (*sql.DB, error) {
	_, err = os.Stat(dbname)
	if os.IsNotExist(err) {
		log.Printf("InitDB: database does not exist %s", dbname)
	}
	db, err = sql.Open("sqlite3", dbname)
	if err != nil {
		return nil, fmt.Errorf("InitDB: Error creating database. %s", err.Error())
	}
	err = syncDB()
	if err != nil {
		return nil, fmt.Errorf("InitDB: Error syncing database. %s", err.Error())
	}
	return db, nil
}

func syncDB() error {
	query := `CREATE TABLE IF NOT EXISTS messages (` +
		`id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,` +
		`uuid char(32) UNIQUE NOT NULL,` +
		`message char(160) NOT NULL,` +
		`mobile char(15) NOT NULL,` +
		`status char(15) NOT NULL,` +
		`retries INTEGER DEFAULT 0,` +
		`created_at TIMESTAMP default CURRENT_TIMESTAMP,` +
		`updated_at TIMESTAMP);`
	_, err = db.Exec(query, nil)
	if err != nil {
		return fmt.Errorf("syncDB: %s", err.Error())
	}
	return nil
}

func InsertMessage(sms *common.SMS) error {
	log.Printf("InsertMessage: %#v", sms)
	stmt, err := db.Prepare("INSERT INTO messages(uuid, message, mobile, status) VALUES(?, ?, ?, ?)")
	defer stmt.Close()
	if err != nil {
		return fmt.Errorf("InsertMessage: Failed to prepare transaction. %s", err.Error())
	}
	_, err = stmt.Exec(sms.UUID, sms.Body, sms.Mobile, sms.Status)
	if err != nil {
		return fmt.Errorf("InsertMessage: Failed to execute transaction. %s", err.Error())
	}
	return nil
}

//TODO: locks for driver.Stmt (stmt) and driver.Conn (db)
func UpdateMessageStatus(sms common.SMS) error {
	log.Printf("Updating msg status %#v", sms)
	stmt, err := db.Prepare("UPDATE messages SET status=?, retries=?, updated_at=DATETIME('now') WHERE uuid=?")
	defer stmt.Close()
	if err != nil {
		return fmt.Errorf("UpdateMessageStatus: %s", err.Error())
	}
	_, err = stmt.Exec(sms.Status, sms.Retries, sms.UUID)
	if err != nil {
		return fmt.Errorf("UpdateMessageStatus: %s", err.Error())
	}
	return nil
}

func GetPendingMessages() ([]common.SMS, error) {
	log.Printf("GetPendingMessages")
	var messages []common.SMS
	query := "SELECT uuid, message, mobile, status, retries FROM" +
		" messages WHERE status != \"sent\" AND retries < 3"
	log.Println("GetPendingMessages: ", query)

	rows, err := db.Query(query)
	if err != nil {
		return messages, fmt.Errorf("GetPendingMessages: %s", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		sms := common.SMS{}
		rows.Scan(&sms.UUID, &sms.Body, &sms.Mobile, &sms.Status, &sms.Retries)
		messages = append(messages, sms)
	}
	//rows.Close()
	return messages, nil
}
