'use strict';
module.exports = {
  validate: function(object, keys) {
    if (object != null) {
      if (keys != null) {
        if (keys instanceof Array) {
          for (var key in keys) {
            if (object[keys[key]] == null) {
              return { valid: false, message: "The value of " + keys[key] + " should not be undefined or null" };
            }
          }
        } else {
          return { valid: false, message: "The keys should be an array" };
        }
      }
    } else {
      return { valid: false, message: "The object should not be undefined or null" };
    }
    return { valid: true };
  }
};