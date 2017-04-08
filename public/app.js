var app = angular.module('authyDemo', []);

app.controller('PhoneVerificationController', function ($scope, $http, $window, $timeout) {

    $scope.setup = {
        via: "sms"
    };
    
    $scope.view = {
        start: true
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

