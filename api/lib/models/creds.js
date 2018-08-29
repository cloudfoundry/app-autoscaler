'use strict';

module.exports = function(sequelize, DataTypes) {
  var CustomMetricsCredentials = sequelize.define('credentials', {
    id: {
      type: DataTypes.STRING,
      field: 'id',
      primaryKey: true
    },
    username: {
      type: DataTypes.STRING,
      field: 'username',
      allowNull: false
    },
    password: {
      type: DataTypes.STRING,
      field: 'password',
      allowNull: false
    }
  },{
    freezeTableName: true,
    timestamps: true,
    createdAt: false,
    updatedAt: 'updated_at'
  });
  return CustomMetricsCredentials;
};
