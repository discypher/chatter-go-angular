var app = angular.module("chatter", []);

app.controller("MainCtrl", ["$scope", function($scope) {
  $scope.messages = [];
  $scope.msg = "";

  var conn = new WebSocket("ws://localhost:3000/ws");

  conn.onclose = function(e) {
    $scope.$apply(function() {
      $scope.messages.push("Exiting");
    })
  }

  conn.onopen = function(e) {
    $scope.$apply(function() {
      $scope.messages.push("Connected");
    })
  }

  conn.onmessage = function(e) {
    $scope.$apply(function() {
      $scope.messages.push(e.data);
    })
  }

  $scope.send = function() {
    if ( typeof $scope.msg !== "undefined" ) {
      conn.send($scope.msg);
    }
    $scope.msg = "";
  }
}])
