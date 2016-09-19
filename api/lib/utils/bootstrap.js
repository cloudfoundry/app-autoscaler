exports.boot = function boot(callback) {
  if(!process.env.DB_URI) {
    callback({ 'message':'DB_URI not found' });
  }
  else if(!process.env.SCHEDULER_URI) {
    callback({ 'message': 'SCHEDULER_URI not found' });
  }
  else{
    require('../../lib/models')(function(error) {
      if (error) {
        callback(error)
      }
      else{
        callback(null);
      }
    });
  }
}
