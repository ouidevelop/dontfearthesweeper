package main

import (
	"database/sql"
	"log"
)

func FindReadyAlerts(sender smsMessager) {
	findReadyAlertStmt, err := DB.Prepare("select ID, PHONE_NUMBER, COUNTRY_CODE, NTH_DAY, TIMEZONE, WEEKDAY from alerts where NEXT_CALL < ?")
	if err != nil {
		log.Fatal(err)
	}
	defer findReadyAlertStmt.Close()

	updateStmt, err := DB.Prepare("UPDATE alerts SET NEXT_CALL = ? WHERE ID = ?;")
	if err != nil {
		log.Fatal(err)
	}
	defer updateStmt.Close()

	nowUTC := Now().Unix()
	rows, err := findReadyAlertStmt.Query(nowUTC)
	if err != nil {
		log.Println("problem exicuting statement: err", err)
	}

	defer rows.Close()
	for rows.Next() {
		var id int
		alert := Alert{}

		err := rows.Scan(&id, &alert.PhoneNumber, &alert.CountryCode, &alert.NthDay, &alert.Timezone, &alert.Weekday)
		if err != nil {
			log.Println("problem scanning rows: err", err)
		}

		nextCall, err := CalculateNextCall(alert)
		if err != nil {
			log.Println("error calculating next call: err", err)
		}

		_, err = updateStmt.Exec(nextCall, id)
		if err != nil {
			log.Println("error exicuting update statement: err", err)
		}

		remind(alert.PhoneNumber, sender)
	}
	if err = rows.Err(); err != nil {
		log.Println("problem iterating through the rows: ", err)
	}
}

func save(alert Alert) error {
	stmt, err := DB.Prepare("INSERT INTO alerts (PHONE_NUMBER, COUNTRY_CODE, NTH_DAY, TIMEZONE, WEEKDAY, NEXT_CALL) VALUES (?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	nextCall, err := CalculateNextCall(alert)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(alert.PhoneNumber, alert.CountryCode, alert.NthDay, alert.Timezone, alert.Weekday, nextCall)
	if err != nil {
		return err
	}
	return nil
}

func startDB(mysqlPassword string) *sql.DB {
	db, err := sql.Open("mysql", mysqlPassword)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTableCommand := `CREATE TABLE IF NOT EXISTS alerts(
				   ID INT NOT NULL AUTO_INCREMENT,
				   PHONE_NUMBER CHAR(10) NOT NULL,
				   COUNTRY_CODE INT NOT NULL,
				   NTH_DAY INT NOT NULL,
				   TIMEZONE VARCHAR(100) NOT NULL,
				   WEEKDAY VARCHAR(20) NOT NULL,
				   NEXT_CALL BIGINT NOT NULL,
				   PRIMARY KEY  (ID)
				)`
	_, err = db.Exec(createTableCommand)

	if err != nil {
		log.Fatal(err)
	}

	return db
}

func removeAlert(alert Alert) error {
	stmt, err := DB.Prepare("DELETE FROM alerts WHERE PHONE_NUMBER = ?;")
	if err != nil {
		return err
	}

	_, err = stmt.Exec(alert.PhoneNumber)
	if err != nil {
		return err
	}
	return nil
}
