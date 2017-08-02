# dontfearthesweeper
Street Sweeping Reminders

site: www.dontfearthesweeper.com

To run the application, you must have go installed. see here: <https://golang.org/doc/install>

I also recommend you use gin to run the application. Gin makes it so that you don't have to restart your server every time you make a change to your code. Once you have Go set up on your computer, you can install gin by running `go get github.com/codegangsta/gin`

When you have gin, you can run the application by going into the project folder and runing `gin`.

You will need some environment variables set. Specifically, you will have to set the following: 
STREETSWEEP_AUTHY_API_KEY - twilio's Authy api key
TWILIO_ID - twilio id
TWILIO_AUTH_TOKEN - twilio authentication token

Once you have the application running, go to localhost:3000 in your browser (or instead of 3000, use whichever port gin tells you to use when you first run gin).


**Testing**

In order to run the tests, `run ginkgo -v` (or `go test` if you don't have ginkgo)

Let me know if you have trouble setting this up, and I'll help you out -mike
