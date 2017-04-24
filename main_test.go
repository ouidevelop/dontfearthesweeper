package main_test

import (
	. "github.com/kurrels/dontfearthesweeper"

	"fmt"
	"time"

	"bytes"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Main", func() {
	Describe("CalculateNextCall", func() {
		It("should calculate the next date to send an alert", func() {
			location, err := time.LoadLocation("America/New_York")
			Expect(err).NotTo(HaveOccurred())

			weekday := 0
			nthWeek := 1
			timezone := "America/New_York"

			nextAlertTime, err := CalculateNextCall(nthWeek, weekday, timezone)
			Expect(err).NotTo(HaveOccurred())
			Expect(nextAlertTime).To(Equal(int64(1494111600))) //2017-05-06 19:00:00 -0400 EDT

			// change the weekday to friday to make sure that the function determines that the next alert is this month
			weekday = 5

			nextAlertTime, err = CalculateNextCall(nthWeek, weekday, timezone)
			fmt.Println("time: ", time.Unix(nextAlertTime, 0).In(location))
			Expect(err).NotTo(HaveOccurred())
			Expect(nextAlertTime).To(Equal(int64(1491519600))) //2017-04-06 19:00:00 -0400 EDT
		})
	})

	Describe("application", func() {
		It("should alert people who's alert is past due", func() {
			err := DB.Ping()
			Expect(err).NotTo(HaveOccurred())

			jsonAlert := []byte(`{"timezone":"America/New_York","times":[{"weekday":0,"nthWeek":1}],"phoneNumber":"8054234224","token":""}`)

			fmt.Println("1")

			clearDB()
			defer clearDB()

			fmt.Println("2")
			req := httptest.NewRequest("POST", "/verification/verify", bytes.NewReader(jsonAlert))
			res := httptest.NewRecorder()
			MockEnv.VerificationVerifyHandler(res, req)
			Expect(res.Code).To(Equal(http.StatusOK))
			fmt.Println("3")

			var nextCall int64
			err = DB.QueryRow("select NEXT_CALL from alerts").Scan(&nextCall)
			Expect(err).NotTo(HaveOccurred())
			Expect(nextCall).To(Equal(int64(1494111600))) //2017-05-06 19:00:00 -0400 EDT
			fmt.Println("4")

			// 1 second after the next call of the alert
			done := MockNow(time.Unix(1494111601, 0))
			defer done()
			fmt.Println("5")

			FindReadyAlerts(MockEnv.MsgSvc)
			expected := &MockMessageService{
				from: "5102414070",
				to:   "8054234224",
				body: "Don't forget about street tomorrow!",
			}
			Expect(MockEnv.MsgSvc).To(Equal(expected))
			fmt.Println("6")
		})
	})
})

func clearDB() {
	_, err := DB.Exec("Truncate table alerts")
	Expect(err).NotTo(HaveOccurred())
}
