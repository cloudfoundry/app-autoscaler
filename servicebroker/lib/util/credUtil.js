'use strict';

var crypto = require('crypto');


module.exports =  function(){

    var credUtils = {};
    credUtils.getUserName = function(){
        var userName = crypto.randomBytes(16).toString('hex');
        return userName;
    },
    credUtils.getPassword = function(){
        var password = crypto.randomBytes(16).toString('hex');
        return password;
    }
    return credUtils;
};