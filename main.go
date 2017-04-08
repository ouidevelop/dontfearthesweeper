package main

import (
	"fmt"
	"os"

	"database/sql"
	"time"

	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/dcu/go-authy"
	_ "github.com/go-sql-driver/mysql"
)

var (
	//All global environment variables should be set at the beginning of the application, then remain unchanged.
	authyAPIKey = os.Getenv("STREETSWEEP_AUTHY_API_KEY")
	Port        = os.Getenv("PORT")
	authyAPI    *authy.Authy
	DB          *sql.DB
)

type StartVerifyReq struct {
	Via         string `json:"via"`
	CountryCode string `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
}

//{"via":"sms","timezone":"America/Phoenix","nth_day":"second","weekday":"friday","country_code":"1","phone_number":"8054234224","token":"3"}

type VerifyCodeReq struct {
	Timezone    string `json:"timezone"`
	NthDay      string `json:"nth_day"`
	Weekday     string `json:"weekday"`
	CountryCode string `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
	Token       string `json:"token"`
}

func init() {
	if authyAPIKey == "" {
		log.Fatal("STREETSWEEP_AUTHY_API_KEY environment variable not set")
	}
	authyAPI = authy.NewAuthyAPI(authyAPIKey)

	db, err := sql.Open("mysql",
		"root:@tcp(127.0.0.1:3306)/hello")
	if err != nil {
		log.Fatal(err)
	}

	DB = db

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		// do something here
	}

	createTableCommand := `CREATE TABLE IF NOT EXISTS alerts(
				   ID INT NOT NULL,
				   PHONE_NUMBER INT NOT NULL,
				   COUNTRY_CODE INT NOT NULL,
				   NTH_DAY INT NOT NULL,
				   TIMEZONE VARCHAR(100) NOT NULL,
				   WEEKDAY VARCHAR(20) NOT NULL,
				   NEXT_CALL BIGINT,
				   PRIMARY KEY  (ID)
				)`
	_, err = DB.Exec(createTableCommand)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/verification/start", verificationStartHandler)
	http.HandleFunc("/verification/verify", verificationVerifyHandler)
	log.Println("Magic happening on port " + Port)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}

func verificationStartHandler(w http.ResponseWriter, r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("1", string(requestDump))

	decoder := json.NewDecoder(r.Body)
	var t StartVerifyReq
	err = decoder.Decode(&t)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	log.Printf("bob: %+v", t)
	countryCodeInt, err := strconv.Atoi(t.CountryCode)
	if err != nil {
		panic(err)
	}
	verification, err := authyAPI.StartPhoneVerification(countryCodeInt, t.PhoneNumber, t.Via, url.Values{})
	if verification.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		//todo: do this better. figure out all the ways that start phone verification could fail and handle all of them well
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "validation code incorrect")
	}
	fmt.Println("verification, err", verification, err)
}

func verificationVerifyHandler(w http.ResponseWriter, r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("2", string(requestDump))

	decoder := json.NewDecoder(r.Body)
	var t VerifyCodeReq
	err = decoder.Decode(&t)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	log.Printf("bob: %+v", t)
	countryCodeInt, err := strconv.Atoi(t.CountryCode)
	if err != nil {
		panic(err)
	}
	verification, err := authyAPI.CheckPhoneVerification(countryCodeInt, t.PhoneNumber, t.Token, url.Values{})
	fmt.Println("verification, err", verification, err)

	if verification.Success {
		w.WriteHeader(200)
		save(t)
	} else {
		//todo: do this better. figure out all the ways that CheckPhoneVerification could fail and handle all of them well
		w.WriteHeader(401)
		io.WriteString(w, "validation code incorrect")
	}
}

//PHONE_NUMBER INT NOT NULL,
//COUNTRY_CODE INT NOT NULL,
//NTH_DAY INT NOT NULL,
//TIMEZONE VARCHAR(100) NOT NULL,
//WEEKDAY VARCHAR(20) NOT NULL,
//NEXT_CALL BIGINT,
//PRIMARY KEY  (ID)

func save(alert VerifyCodeReq) {
	stmt, err := DB.Prepare("INSERT INTO alerts (PHONE_NUMBER, COUNTRY_CODE, NTH_DAY, TIMEZONE, WEEKDAY, NEXT_CALL) VALUES (?,?,?,?,?,?)")
	if err != nil {
		log.Fatal(err) //todo: change these log.fatals to a more reasonable error handling
	}

	nextCall, err := CalculateNextCall(alert)
	if err != nil {
		log.Fatal(err)
	}

	phoneNumber, err := strconv.Atoi(alert.PhoneNumber)
	if err != nil {
		log.Fatal(err)
	}

	countryCode, err := strconv.Atoi(alert.CountryCode)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(phoneNumber, countryCode, alert.NthDay, alert.Timezone, alert.Weekday, nextCall)
	if err != nil {
		log.Fatal(err)
	}
}

var Now = func() time.Time {
	return time.Now()
}

func CalculateNextCall(alert VerifyCodeReq) (int64, error) {
	var NextCallTime int64
	weekday, err := strconv.Atoi(alert.Weekday)
	if err != nil {
		fmt.Println("problem converting weekday string to int")
		return NextCallTime, err
	}
	nthDay, err := strconv.Atoi(alert.NthDay)
	if err != nil {
		fmt.Println("problem converting weekday string to int")
		return NextCallTime, err
	}

	location, err := time.LoadLocation(alert.Timezone)
	if err != nil {
		return NextCallTime, err
	}

	now := Now().In(location)
	timeAtNthDayOfMonth := TimeAtNthDayOfMonth(now, nthDay, weekday, 19) //todo: change this hard coded hour
	if now.After(timeAtNthDayOfMonth) {
		timeAtNthDayOfMonth = TimeAtNthDayOfMonth(now.AddDate(0, 1, 0), nthDay, weekday, 19)
	}

	NextCallTime = timeAtNthDayOfMonth.Unix()

	return NextCallTime, nil
}

func TimeAtNthDayOfMonth(t time.Time, nthDay int, weekday int, hour int) time.Time {

	firstDayOfThisMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	dateOfFirstWeekday := ((weekday+7)-int(firstDayOfThisMonth.Weekday()))%7 + 1
	fmt.Println("dateOfFirstWeekday ", dateOfFirstWeekday)
	dateOfNthWeekday := dateOfFirstWeekday + ((nthDay - 1) * 7)
	fmt.Println("nthday ", dateOfNthWeekday)
	TimeAtNthDayOfMonth := time.Date(t.Year(), t.Month(), dateOfNthWeekday, hour, 0, 0, 0, t.Location())
	return TimeAtNthDayOfMonth
}
