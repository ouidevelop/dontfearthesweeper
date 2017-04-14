package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"database/sql"
	"github.com/dcu/go-authy"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sfreiberg/gotwilio"
)

const from = "+15102414070"

var (
	//All global environment variables should be set at the beginning of the application, then remain unchanged.
	authyAPIKey     = os.Getenv("STREETSWEEP_AUTHY_API_KEY")
	twilioID        = os.Getenv("TWILIO_ID")
	twilioAuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	port            = os.Getenv("PORT")
	mysqlPassword   = os.Getenv("MYSQL_PASSWORD")

	twilio   = gotwilio.NewTwilioClient(twilioID, twilioAuthToken)
	authyAPI = authy.NewAuthyAPI(authyAPIKey)
	db       *sql.DB
)

type StartVerification struct {
	Via         string `json:"via"`
	CountryCode int    `json:"country_code"`
	PhoneNumber int    `json:"phone_number"`
}

type Alert struct {
	Timezone    string `json:"timezone"`
	NthDay      int    `json:"nth_day"`
	Weekday     int    `json:"weekday"`
	CountryCode int    `json:"country_code"`
	PhoneNumber int    `json:"phone_number"`
	Token       string `json:"token"`
}

func init() {
	if authyAPIKey == "" {
		log.Fatal("STREETSWEEP_AUTHY_API_KEY environment variable not set")
	}
	if twilioID == "" {
		log.Fatal("TWILIO_ID environment variable not set")
	}
	if twilioAuthToken == "" {
		log.Fatal("TWILIO_AUTH_TOKEN environment variable not set")
	}
	if port == "" {
		port = "8080"
	}
	if mysqlPassword == "" {
		log.Fatal("MYSQL_PASSWORD environment variable not set")
	}

	db = startDB()
}

func main() {

	go func() {
		for range time.Tick(10 * time.Second) {
			FindReadyAlerts()
		}
	}()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/verification/start", verificationStartHandler)
	http.HandleFunc("/verification/verify", VerificationVerifyHandler)
	log.Println("Magic happening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// for mocking:
var StartPhoneVerification = func (countryCode int, phoneNumber string, via string, params url.Values) (*authy.PhoneVerificationStart, error) {
	return authyAPI.StartPhoneVerification(countryCode, phoneNumber, via, params)
}

func verificationStartHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var t StartVerification
	err := decoder.Decode(&t)
	if err != nil {
		log.Println("error decoding json: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "oops! we made a mistake")
		return
	}
	defer r.Body.Close()
	t.CountryCode = 1

	verification, err := StartPhoneVerification(t.CountryCode, strconv.Itoa(t.PhoneNumber), t.Via, url.Values{})
	if !verification.Success {
		//todo: do this better. figure out all the ways that start phone verification could fail and handle all of them well
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "problem starting phone verification")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// for mocking:
var CheckPhoneVerification = func (countryCode int, phoneNumber string, verificationCode string, params url.Values) (*authy.PhoneVerificationCheck, error) {
	return authyAPI.CheckPhoneVerification(countryCode, phoneNumber, verificationCode, params)
}

func VerificationVerifyHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var t Alert
	err := decoder.Decode(&t)
	if err != nil {
		log.Println("error decoding json: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "oops! we made a mistake")
		return
	}
	defer r.Body.Close()
	t.CountryCode = 1

	verification, err := CheckPhoneVerification(t.CountryCode, strconv.Itoa(t.PhoneNumber), t.Token, url.Values{})
	if !verification.Success {
		//todo: do this better. figure out all the ways that CheckPhoneVerification could fail and handle all of them well
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "validation code incorrect")
		return
	}

	err = save(t)
	if err != nil {
		log.Println("problem saving new alert to database: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "oops! we made a mistake")
		return
	}

	w.WriteHeader(http.StatusOK)
}

var Now = func() time.Time {

	return time.Now()
}

func CalculateNextCall(alert Alert) (int64, error) {

	var NextCallTime int64

	location, err := time.LoadLocation(alert.Timezone)
	if err != nil {
		return NextCallTime, err
	}

	now := Now().In(location)
	t := timeAtNthDayOfMonth(now, alert.NthDay, alert.Weekday, 19) //todo: change this hard coded hour
	if now.After(t) {
		t = timeAtNthDayOfMonth(now.AddDate(0, 1, 0), alert.NthDay, alert.Weekday, 19)
	}

	NextCallTime = t.Unix()

	return NextCallTime, nil
}

func timeAtNthDayOfMonth(t time.Time, nthDay int, weekday int, hour int) time.Time {

	firstDayOfThisMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	dateOfFirstWeekday := ((weekday+7)-int(firstDayOfThisMonth.Weekday()))%7 + 1
	dateOfNthWeekday := dateOfFirstWeekday + ((nthDay - 1) * 7)
	TimeAtNthDayOfMonth := time.Date(t.Year(), t.Month(), dateOfNthWeekday, hour, 0, 0, 0, t.Location())
	return TimeAtNthDayOfMonth
}

func remind(countryCode, phoneNumber int) {

	twilioNumber := "+" + strconv.Itoa(countryCode) + strconv.Itoa(phoneNumber)
	fmt.Println("sending message")
	message := "Don't forget about street tomorrow!"
	resp, exception, err := twilio.SendSMS(from, twilioNumber, message, "", "")
	fmt.Println("to: ", twilioNumber, "resp: ", resp, "exception: ", exception, "err: ", err)
}
