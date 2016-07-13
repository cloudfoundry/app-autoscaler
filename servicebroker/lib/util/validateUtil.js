'use strict';
module.exports = {
  validate: function(object, keys) {
    if (object != null) {
      if (keys != null) {
        if (keys instanceof Array) {
          for (var key in keys) {
            if (typeof (object[keys[key]]) === "undefined" || object[keys[key]] === null) {
              return false;
            }
          }
        } else {
          return false;
        }
      }
    } else {
      return false;
    }
    return true;
  }
};