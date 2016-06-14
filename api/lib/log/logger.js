// TODO: To decide and adopt to some logging utility like winston
module.exports = {
  info: function(message) {
    console.log(message);
  },
  warn: function(message) {
    console.log(message);
  },
  error: function(message, error) {
    console.error(message, error);
  },
  log: function(message) {
    console.log(message);
  }
};
