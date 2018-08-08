// placeholder for future logging enhancement
var util = require('util');

module.exports = {
  info: function(message,context) {	
	var logItem = {
	  timestamp: new Date(),
	  source: 'autoscaler:servicebroker',
	  text: message,
	  log_level: 'info',
	  data: context === null ? {} : context
	};
	console.log (util.format('%j', logItem)); 
  },
  error: function(message, error) {
    var logItem = {
      timestamp: new Date(),
      source: 'autoscaler:servicebroker',
      text: message,
      log_level: 'error',
      data: error
    };
    console.log (util.format('%j', logItem));
  }
};
