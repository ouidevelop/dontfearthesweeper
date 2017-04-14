var app = angular.module('authyDemo', []);

app.controller('PhoneVerificationController', function ($scope, $http, $window, $timeout) {

    $scope.setup = {
        via: "sms"
    };

    $scope.setTimeZone = function(zone) {
        $scope.setup.timezone = zone;
    };

    $scope.setWeek = function(n) {
        $scope.setup.nth_day = n;
    };

    $scope.setDay = function(day) {
        $scope.setup.weekday = day;
    };

    $scope.setMethod = function(method) {
        $scope.setup.via = method;
    };

    /**
     * Initialize Phone Verification
     */
    $scope.startVerification = function () {
        console.log("scope.setup", $scope.setup)
        $http.post('/verification/start', $scope.setup)
            .success(function (data, status, headers, config) {
                console.log("Verification started: ", data);
            })
            .error(function (data, status, headers, config) {
                console.error("Phone verification error: ", data);
            });
    };

    /**
     * Verify phone token
     */
    $scope.verifyToken = function () {
        $scope.setup.nth_day = parseInt($scope.setup.nth_day, 10)
        $scope.setup.weekday = parseInt($scope.setup.weekday, 10)
        console.log("scope.setup", $scope.setup)
        $http.post('/verification/verify', $scope.setup)
            .success(function (data, status, headers, config) {
                console.log("Phone Verification Success success: ", data);
            })
            .error(function (data, status, headers, config) {
                console.error("Verification error: ", data);
            });
    };
});

