// TODO: To decide and adopt to some logging utility like winston
var util = require('util');

module.exports = {
  info: function(message,context) {
    if(context) {
      console.log(message, util.format('%j', context));
    }
    else {
      console.log(message);
    }
    
  },
  error: function(message, error) {
    console.error(message, util.format('%j', error));
  }
};
