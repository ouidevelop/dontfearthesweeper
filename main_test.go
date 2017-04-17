package main_test

import (
	. "github.com/kurrels/dontfearthesweeper"

	"fmt"
	"time"

	"net/http"
	"net/http/httptest"

	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Main", func() {
	Describe("CalculateNextCall", func() {
		It("should calculate the next date to send an alert", func() {
			location, err := time.LoadLocation("America/New_York")
			Expect(err).NotTo(HaveOccurred())

			alert := Alert{
				Timezone:    "America/New_York",
				NthDay:      1,
				Weekday:     0,
				CountryCode: 1,
				PhoneNumber: 8054234224,
			}

			nextAlertTime, err := CalculateNextCall(alert)
			fmt.Println("time: ", time.Unix(nextAlertTime, 0).In(location))
			Expect(err).NotTo(HaveOccurred())
			Expect(nextAlertTime).To(Equal(int64(1494111600))) //2017-05-06 19:00:00 -0400 EDT

			// change the weekday to friday to make sure that the function determines that the next alert is this month
			alert.Weekday = 5

			nextAlertTime, err = CalculateNextCall(alert)
			fmt.Println("time: ", time.Unix(nextAlertTime, 0).In(location))
			Expect(err).NotTo(HaveOccurred())
			Expect(nextAlertTime).To(Equal(int64(1491519600))) //2017-04-06 19:00:00 -0400 EDT
		})
	})

	Describe("application", func() {
		It("should alert people who's alert is past due", func() {

			alert := Alert{
				Timezone:    "America/New_York",
				NthDay:      1,
				Weekday:     0,
				CountryCode: 1,
				PhoneNumber: 8054234224,
			}

			jsonAlert, err := json.Marshal(alert)
			Expect(err).NotTo(HaveOccurred())

			clearDB()
			defer clearDB()

			req := httptest.NewRequest("POST", "/verification/verify", bytes.NewReader(jsonAlert))
			res := httptest.NewRecorder()
			MockEnv.VerificationVerifyHandler(res, req)
			Expect(res.Code).To(Equal(http.StatusOK))

			var nextCall int64
			err = DB.QueryRow("select NEXT_CALL from alerts").Scan(&nextCall)
			Expect(err).NotTo(HaveOccurred())
			Expect(nextCall).To(Equal(int64(1494111600))) //2017-05-06 19:00:00 -0400 EDT

			// 1 second after the next call of the alert
			done := MockNow(time.Unix(1494111601, 0))
			defer done()

			FindReadyAlerts(MockEnv.MsgSvc)
			expected := &MockMessageService{
				from: "5102414070",
				to:   "8054234224",
				body: "Don't forget about street tomorrow!",
			}
			Expect(MockEnv.MsgSvc).To(Equal(expected))
		})
	})
})

func clearDB() {
	_, err := DB.Exec("Truncate table alerts")
	Expect(err).NotTo(HaveOccurred())
}
