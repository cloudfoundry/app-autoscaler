'use strict';
module.exports = {
  isUndefined: function(value) {
    return typeof(value) === "undefined";
  },

  isNull: function(value) {
    return !value && typeof(value) != "undefined" && value != 0;
  },

  validate: function(object, keys) {
    if (this.isNull(object) === false && this.isUndefined(object) === false) {
      if (this.isNull(keys) === false && this.isUndefined(keys) === false) {
        if (keys instanceof Array) {
          for (var key in keys) {
            if (this.isUndefined(object[keys[key]]) === true || this.isNull(object[keys[key]]) === true) {
              return { valid: false, msg: 'The value of ' + key + ' should not be null or undefined' };
            }
          }
        } else {
          return { valid: false, msg: 'The keys should be not defined or an array' };
        }
      }
    } else {
      return { valid: false, msg: 'The test object should not be null or undefined' };
    }
    return { valid: true };
  }
};