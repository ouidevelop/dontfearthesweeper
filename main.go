package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"database/sql"

	"github.com/dcu/go-authy"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sfreiberg/gotwilio"
)

type Env struct {
	MsgSvc MessageServicer
}

const from = "5102414070"

var (
	//All global environment variables should be set at the beginning of the application, then remain unchanged.
	DB *sql.DB
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
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	if mysqlPassword == "" {
		log.Fatal("MYSQL_PASSWORD environment variable not set")
	}
	DB = startDB(mysqlPassword)
}

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	twilioID := os.Getenv("TWILIO_ID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	if twilioID == "" {
		log.Fatal("TWILIO_ID environment variable not set")
	}
	if twilioAuthToken == "" {
		log.Fatal("TWILIO_AUTH_TOKEN environment variable not set")
	}

	authyAPIKey := os.Getenv("STREETSWEEP_AUTHY_API_KEY")
	if authyAPIKey == "" {
		log.Fatal("STREETSWEEP_AUTHY_API_KEY environment variable not set")
	}

	msgSvc := TwilioMessageService{
		twilio: gotwilio.NewTwilioClient(twilioID, twilioAuthToken),
		authy:  authy.NewAuthyAPI(authyAPIKey),
	}

	env := Env{
		MsgSvc: &msgSvc,
	}

	go func() {
		for range time.Tick(10 * time.Second) {
			FindReadyAlerts(env.MsgSvc)
		}
	}()

	isProduction := os.Getenv("STREETSWEEP_PRODUCTION")
	if isProduction == "true" {
		go func() {
			for range time.Tick(5 * time.Minute) {
				resp, err := http.Get("https://dontfearthesweeper.herokuapp.com/")
				if err != nil {
					log.Println("problem pinging website: ", err)
				}
				if resp.StatusCode != http.StatusOK {
					log.Println("non-200 status code from healthcheck: ", resp.Status)
				}
			}
		}()
	}

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/verification/start", env.verificationStartHandler)
	http.HandleFunc("/verification/verify", env.VerificationVerifyHandler)
	http.HandleFunc("/alerts/stop", env.stopAlertHandler)
	http.HandleFunc("/alerts/bob", env.bob)
	log.Println("Magic happening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (env *Env) bob(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	time.Sleep(35 * time.Second)
	fmt.Println("bobbobobobobobobobobobobobo")
}

func (env *Env) stopAlertHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("in stopAlertHandler")
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
	log.Println("alert: ", t)

	verification, err := env.MsgSvc.VerifyCode(strconv.Itoa(t.PhoneNumber), t.Token)
	if !verification {
		log.Println("verification attempt not successful: error: ", err)
		//todo: do this better. figure out all the ways that CheckPhoneVerification could fail and handle all of them well
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "validation code incorrect")
		return
	}

	err = removeAlerts(t)
	if err != nil {
		log.Println("problem deleting alert to database: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "oops! we made a mistake")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (env *Env) verificationStartHandler(w http.ResponseWriter, r *http.Request) {

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

	verified, err := env.MsgSvc.RequestCode(strconv.Itoa(t.PhoneNumber))
	if !verified {
		//todo: do this better. figure out all the ways that start phone verification could fail and handle all of them well
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "problem starting phone verification")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (env *Env) VerificationVerifyHandler(w http.ResponseWriter, r *http.Request) {

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

	verified, err := env.MsgSvc.VerifyCode(strconv.Itoa(t.PhoneNumber), t.Token)
	if !verified {
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
	t := timeAtNthDayOfMonth(now, alert.NthDay, alert.Weekday, 19).Add(-24 * time.Hour) //todo: change this hard coded hour
	if now.After(t) {
		t = timeAtNthDayOfMonth(now.AddDate(0, 1, 0), alert.NthDay, alert.Weekday, 19).Add(-24 * time.Hour)
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

func remind(phoneNumber int, sender smsMessager) {

	fmt.Println("sending message")
	message := "Don't forget about street tomorrow!"
	err := sender.Send(from, strconv.Itoa(phoneNumber), message)
	if err != nil {
		log.Println("problem sending message: ", err)
	}
}
