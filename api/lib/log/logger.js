// TODO: To decide and adopt to some logging utility like winston
module.exports = {
  info: function(message,context) {
    if(context) {
      console.log(message,JSON.stringify(context, null, 2));
    }
    else {
      console.log(message);
    }
    
  },
  error: function(message, error) {
    console.error(message, JSON.stringify(error, null, 2));
  }
};
