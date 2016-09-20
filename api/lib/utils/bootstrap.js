var async = require ('async');

exports.boot = function boot(cb) {
  async.series([
    function(callback) {
      if (!process.env.DB_URI) {
        callback({ 'message':'DB_URI not found' });		
      } 
      else {
        callback(null);
      }
    },
    function(callback) {
      if (!process.env.SCHEDULER_URI) {
        callback({ 'message':'SCHEDULER_URI not found' });		
      } 
      else {
        callback (null);
      }
    },
    function(callback) {
      require('../../lib/models')(function(error) {
        if (error) {
          callback(error)
        } 
        else {
          callback(null);
        }
      });
    }
  ],
  function(err, results) {
    cb (err);
  });
}
