'use strict';
var bcrypt = require('bcrypt-nodejs');
module.exports = function(sequelize, DataTypes) {
  var credentials = sequelize.define('credentials', {
    bindingId: {
      type: DataTypes.STRING,
      field: 'binding_id',
      allowNull: false
    },
    username: {
      type: DataTypes.STRING,
      field: 'username',
      primaryKey: true,
      allowNull: false
    },
    password: {
        type: DataTypes.STRING,
        field: 'password',
        allowNull: false
    }
  }, {
    classMethods: {
      associate: function(models) {
        credentials.belongsTo(models.binding, { foreignKey: 'binding_id',onDelete: "CASCADE",onUpdate:"CASCADE"});
      },
      generateHash: function(password){
        return bcrypt.hashSync(password, bcrypt.genSaltSync(8));
      }
    },
    freezeTableName: true,
    timestamps: false
  });
  return credentials;
};