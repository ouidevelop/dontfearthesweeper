package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/dcu/go-authy"
	"strconv"
)

var (
	//All global environment variables should be set at the beginning of the application, then remain unchanged.
	authyAPIKey = os.Getenv("STREETSWEEP_AUTHY_API_KEY")
	Port        = os.Getenv("PORT")
	authyAPI    *authy.Authy
)

type StartVerifyReq struct {
	Via         string `json:"via"`
	CountryCode string    `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
}

type VerifyCodeReq struct {
	Token       string `json:"token"`
	CountryCode string    `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
}

func init() {
	if authyAPIKey == "" {
		log.Fatal("STREETSWEEP_AUTHY_API_KEY environment variable not set")
	}
	authyAPI = authy.NewAuthyAPI(authyAPIKey)
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/verification/start", verificationStartHander)
	http.HandleFunc("/verification/verify", verificationVerifyHander)
	log.Println("Magic happening on port " + Port)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}

// webHookHandler handles events from GitHub. It first verifies that the request came from GitHub, then if the event
// is a push event, it passes the event on to processPushEvent.
func verificationStartHander(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t StartVerifyReq
	err := decoder.Decode(&t)
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
	fmt.Println("verification, err", verification, err)
}

func verificationVerifyHander(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t VerifyCodeReq
	err := decoder.Decode(&t)
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
}
